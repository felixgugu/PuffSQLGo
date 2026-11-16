import assert from 'node:assert/strict';
import { afterEach, describe, test } from 'node:test';
import {
  currentDialect,
  getDialect,
  mssqlDialect,
  mysqlDialect,
  setCurrentDialect,
} from '../src/dialects';
import { generateBatchChangesScript } from '../src/utils/batchChangeGenerator';
import {
  escapeIdentifier,
  formatSqlLiteral,
  formatTableName,
  generateDeleteStatement,
  generateInsertStatement,
  generateUpdateStatement,
  parseTargetTableFromSql,
} from '../src/utils/sqlGenerator';
import { wrapIdentifierIfNeeded } from '../src/utils/sqlParser';
import { splitSqlBatches, splitSqlStatements } from '../src/utils/sqlStatementExtractor';
import { generateColumnSelectSql, generateTableSelectSql } from '../src/utils/tableSelectSql';
import { MYSQL_SQL_FUNCTIONS, MYSQL_SQL_KEYWORDS, MYSQL_SQL_SNIPPETS } from '../src/utils/sqlKeywordsMySql';

/** Drops the generated timestamp header so statements can be compared exactly. */
function body(sql: string): string {
  return sql.split('\n').slice(1).join('\n');
}

describe('dialects', () => {
  afterEach(() => setCurrentDialect(mssqlDialect));

  test('unknown or missing engines resolve to SQL Server, the default dialect', () => {
    assert.equal(getDialect(undefined), mssqlDialect);
    assert.equal(getDialect(''), mssqlDialect);
    assert.equal(getDialect('mysql'), mysqlDialect);
    assert.equal(currentDialect(), mssqlDialect);
  });

  test('MySQL quotes identifiers with backticks and qualifies tables by database', () => {
    assert.equal(escapeIdentifier('order`s', mysqlDialect), '`order``s`');
    assert.equal(formatTableName('orders', 'shop', 'shop', mysqlDialect), '`shop`.`orders`');
    // The backend reports the database as the schema; either one alone is enough.
    assert.equal(formatTableName('orders', 'shop', undefined, mysqlDialect), '`shop`.`orders`');
    assert.equal(formatTableName('orders', undefined, undefined, mysqlDialect), '`orders`');
    assert.equal(wrapIdentifierIfNeeded('order items', mysqlDialect), '`order items`');
    assert.equal(wrapIdentifierIfNeeded('order', mysqlDialect), '`order`');
    assert.equal(wrapIdentifierIfNeeded('orders', mysqlDialect), 'orders');
  });

  test('SQL Server output is unchanged when the dialect is passed explicitly', () => {
    assert.equal(escapeIdentifier('a]b', mssqlDialect), '[a]]b]');
    assert.equal(formatTableName('Orders', undefined, 'Sales', mssqlDialect), '[Sales].[dbo].[Orders]');
    assert.equal(formatSqlLiteral("O'Brien", mssqlDialect), "N'O''Brien'");
    assert.equal(formatSqlLiteral('9223372036854775807', mssqlDialect, 'BIGINT'), "N'9223372036854775807'");
  });

  test('MySQL string literals escape quotes and backslashes and carry no N prefix', () => {
    assert.equal(formatSqlLiteral("O'Brien", mysqlDialect), "'O''Brien'");
    assert.equal(formatSqlLiteral('C:\\temp', mysqlDialect), "'C:\\\\temp'");
    assert.equal(formatSqlLiteral(true, mysqlDialect), '1');
    assert.equal(formatSqlLiteral(null, mysqlDialect), 'NULL');
  });

  test('MySQL writes numeric text back as a number only for numeric columns', () => {
    assert.equal(formatSqlLiteral('9223372036854775807', mysqlDialect, 'BIGINT'), '9223372036854775807');
    assert.equal(formatSqlLiteral('18446744073709551615', mysqlDialect, 'UNSIGNED BIGINT'), '18446744073709551615');
    assert.equal(formatSqlLiteral('12.50', mysqlDialect, 'DECIMAL'), '12.50');
    // Text columns keep their quotes, and so does anything that is not a plain number.
    assert.equal(formatSqlLiteral('00123', mysqlDialect, 'VARCHAR'), "'00123'");
    assert.equal(formatSqlLiteral('00123', mysqlDialect), "'00123'");
    assert.equal(formatSqlLiteral('1 OR 1=1', mysqlDialect, 'BIGINT'), "'1 OR 1=1'");
    assert.equal(formatSqlLiteral('1e5', mysqlDialect, 'DOUBLE'), "'1e5'");
  });

  test('MySQL single-row DML is capped with LIMIT 1 instead of a @@ROWCOUNT guard', () => {
    const params = {
      tableName: 'orders',
      schema: 'shop',
      database: 'shop',
      columns: [
        { name: 'id', dataType: 'BIGINT', isPrimaryKey: true, isIdentity: true },
        { name: 'note', dataType: 'VARCHAR' },
      ],
      row: ['9007199254740993', "it's"],
      primaryKeyColumns: ['id'],
      dialect: mysqlDialect,
    };
    assert.equal(
      body(generateUpdateStatement(params)),
      "UPDATE `shop`.`orders`\nSET\n  `note` = 'it''s'\nWHERE `id` = 9007199254740993\nLIMIT 1;"
    );
    assert.equal(
      body(generateDeleteStatement(params)),
      'DELETE FROM `shop`.`orders`\nWHERE `id` = 9007199254740993\nLIMIT 1;'
    );
    assert.equal(
      body(generateInsertStatement(params)),
      "INSERT INTO `shop`.`orders` (`note`)\nVALUES ('it''s');"
    );
    assert.equal(
      body(generateInsertStatement({ ...params, columns: [params.columns[0]!], row: ['1'] })),
      'INSERT INTO `shop`.`orders` () VALUES ();'
    );
  });

  test('MySQL grid commit runs between START TRANSACTION and COMMIT with no T-SQL', () => {
    const sql = generateBatchChangesScript({
      tableName: 'orders',
      schema: 'shop',
      database: 'shop',
      dialect: mysqlDialect,
      columnTypes: { id: 'BIGINT', total: 'DECIMAL', note: 'VARCHAR' },
      deletes: [{ pkWhere: { id: '9007199254740993' } }],
      inserts: [{ values: [{ column: 'note', value: 'new' }] }, { values: [] }],
      updates: [{ rowIndex: 4, pkWhere: { id: 7 }, updates: { total: '12.50', note: null } }],
    });
    const lines = body(sql);
    assert.equal(
      lines,
      [
        'START TRANSACTION;',
        '  -- [刪除 1] 依主鍵刪除 1 列',
        '  DELETE FROM `shop`.`orders`',
        '  WHERE `id` = 9007199254740993',
        '  LIMIT 1;',
        '',
        '  -- [新增 1] 寫入 1 個欄位',
        '  INSERT INTO `shop`.`orders` (`note`)',
        "  VALUES ('new');",
        '',
        '  -- [新增 2] 以資料表預設值新增 1 列',
        '  INSERT INTO `shop`.`orders` () VALUES ();',
        '',
        '  -- [列 5] 更新 2 個欄位',
        '  UPDATE `shop`.`orders`',
        '  SET `total` = 12.50, `note` = NULL',
        '  WHERE `id` = 7',
        '  LIMIT 1;',
        '',
        'COMMIT;',
      ].join('\n')
    );
    assert.doesNotMatch(sql, /@@|THROW|BEGIN TRY|N'|\[shop\]|\[orders\]/);
  });

  test('the SQL Server grid commit keeps its @@ROWCOUNT guards', () => {
    const sql = generateBatchChangesScript({
      tableName: 'Orders',
      updates: [{ rowIndex: 0, pkWhere: { Id: 1 }, updates: { Note: 'x' } }],
    });
    assert.match(sql, /UPDATE \[dbo\]\.\[Orders\]\n  SET \[Note\] = N'x'\n  WHERE \[Id\] = 1;\n\n  IF @@ROWCOUNT <> 1 THROW 50001/);
    assert.match(sql, /BEGIN TRY[\s\S]*COMMIT TRANSACTION;[\s\S]*END CATCH;$/);
  });

  test('generated SELECT statements drop the schema for MySQL', () => {
    assert.equal(generateTableSelectSql('shop', 'order items', mysqlDialect), 'SELECT * FROM `order items`;\n');
    assert.equal(generateColumnSelectSql('shop', 'orders', 'key', mysqlDialect), 'SELECT `key` FROM orders;\n');
    assert.equal(generateTableSelectSql('dbo', 'Orders', mssqlDialect), 'SELECT * FROM dbo.Orders;\n');
    assert.equal(mysqlDialect.selectLimitSql('`shop`.`orders`', 500), 'SELECT * FROM `shop`.`orders` LIMIT 500;');
    assert.equal(mssqlDialect.selectLimitSql('[dbo].[Orders]', 500), 'SELECT TOP 500 * FROM [dbo].[Orders];');
  });

  test('target table parsing understands backtick-quoted names', () => {
    assert.deepEqual(parseTargetTableFromSql('SELECT * FROM `shop`.`order items` WHERE id = 1'), {
      schema: 'shop',
      tableName: 'order items',
    });
    assert.deepEqual(parseTargetTableFromSql('SELECT * FROM [Sales].[dbo].[Orders]'), {
      schema: 'dbo',
      tableName: 'Orders',
    });
  });

  test('statement splitting follows the active dialect', () => {
    const script = "SELECT 'a;b' AS `x;y`; # note; with 'quote\nSELECT 'it\\'s; fine';\nGO\nSELECT 3;";

    setCurrentDialect(mysqlDialect);
    assert.deepEqual(splitSqlStatements("SELECT 'a;b' AS `x;y`; # note; with 'quote\nSELECT 'it\\'s; fine'"), [
      "SELECT 'a;b' AS `x;y`",
      "# note; with 'quote\nSELECT 'it\\'s; fine'",
    ]);
    // GO is not a batch separator in MySQL: the script stays one batch.
    assert.equal(splitSqlBatches(script).length, 1);

    setCurrentDialect(mssqlDialect);
    assert.deepEqual(splitSqlBatches('SELECT 1;\nGO\nSELECT 2;'), ['SELECT 1;', 'SELECT 2;']);
    assert.deepEqual(splitSqlStatements('SELECT [a;b] FROM t; SELECT 2'), ['SELECT [a;b] FROM t', 'SELECT 2']);
  });

  test('MySQL routine helpers use information_schema, SHOW CREATE and CALL', () => {
    assert.match(mysqlDialect.routineListSql("sh'op"), /FROM information_schema\.ROUTINES WHERE ROUTINE_SCHEMA = 'sh''op'/);
    assert.equal(mysqlDialect.definitionSql('shop', 'shop', 'v_orders', 'VIEW'), 'SHOW CREATE VIEW `shop`.`v_orders`;');
    assert.match(mysqlDialect.callProcedureSql('shop', 'shop', 'refresh'), /CALL `shop`\.`refresh`\(\);/);
    assert.match(mssqlDialect.callProcedureSql('Sales', 'dbo', 'Refresh'), /USE \[Sales\];\nGO\n\nEXEC \[dbo\]\.\[Refresh\];/);
  });

  test('SQL Server-only features are switched off for MySQL only', () => {
    for (const enabled of Object.values(mssqlDialect.capabilities)) assert.equal(enabled, true);
    for (const enabled of Object.values(mysqlDialect.capabilities)) assert.equal(enabled, false);
  });

  test('MySQL completion lists drop T-SQL-only entries and add MySQL ones', () => {
    const keywords = MYSQL_SQL_KEYWORDS.map((item) => item.name);
    const functions = MYSQL_SQL_FUNCTIONS.map((item) => item.name);
    const snippets = MYSQL_SQL_SNIPPETS.map((item) => item.label);
    assert.ok(keywords.includes('LIMIT') && keywords.includes('SELECT'));
    assert.ok(!keywords.includes('TOP') && !keywords.includes('WITH (NOLOCK)'));
    assert.ok(functions.includes('IFNULL') && functions.includes('COUNT'));
    assert.ok(!functions.includes('ISNULL') && !functions.includes('GETDATE'));
    assert.equal(functions.filter((name) => name === 'DATEDIFF').length, 1);
    assert.ok(snippets.includes('sellimit') && !snippets.includes('seltop'));
  });
});
