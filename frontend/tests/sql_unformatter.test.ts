import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import { unformatSql } from '../src/utils/sqlUnformatter';

describe('sql_unformatter', () => {
  test('collapses newlines and runs of whitespace between SQL tokens', () => {
    const sql = `SELECT
  ProductId,
  ProductName
FROM dbo.Products
WHERE IsActive = 1`;

    assert.equal(
      unformatSql(sql),
      'SELECT ProductId, ProductName FROM dbo.Products WHERE IsActive = 1'
    );
  });

  test('preserves whitespace inside string literals', () => {
    assert.equal(
      unformatSql("SELECT\n  'USB-C  集線器'\n  AS Name"),
      "SELECT 'USB-C  集線器' AS Name"
    );
    assert.equal(
      unformatSql("SELECT 'line 1\n  line 2' AS x"),
      "SELECT 'line 1\n  line 2' AS x"
    );
  });

  test('handles escaped quotes without ending the literal early', () => {
    assert.equal(
      unformatSql("SELECT  'it''s   fine'  AS x"),
      "SELECT 'it''s   fine' AS x"
    );
  });

  test('preserves bracket and double-quoted identifiers', () => {
    assert.equal(
      unformatSql('SELECT [My  Table].[Col  1], "My  Column" FROM [My  Table]'),
      'SELECT [My  Table].[Col  1], "My  Column" FROM [My  Table]'
    );
    assert.equal(
      unformatSql('SELECT [A]]B] FROM t'),
      'SELECT [A]]B] FROM t'
    );
    assert.equal(
      unformatSql('SELECT "A""B" FROM t'),
      'SELECT "A""B" FROM t'
    );
  });

  test('keeps line comments and their terminating newline', () => {
    assert.equal(
      unformatSql('SELECT 1 -- keep   spaces\nFROM t'),
      'SELECT 1 -- keep   spaces\nFROM t'
    );
  });

  test('collapses whitespace inside block comments but keeps the comment', () => {
    assert.equal(
      unformatSql('SELECT /* a\n   b */  1'),
      'SELECT /* a b*/ 1'
    );
    assert.equal(
      unformatSql('SELECT /* outer /* inner\ncomment */ end */ 1'),
      'SELECT /* outer /* inner comment */ end*/ 1'
    );
  });

  test('treats comment markers inside strings as literal text', () => {
    assert.equal(
      unformatSql("SELECT '-- not a comment', '/* not a block */'"),
      "SELECT '-- not a comment', '/* not a block */'"
    );
  });

  test('keeps GO batch separators on their own lines', () => {
    assert.equal(
      unformatSql('SELECT 1\nGO\nSELECT 2'),
      'SELECT 1\nGO\nSELECT 2'
    );
    assert.equal(
      unformatSql('SELECT 1\nGO 2 -- batch\nSELECT 2'),
      'SELECT 1\nGO 2 -- batch\nSELECT 2'
    );
  });

  test('does not treat GO inside a comment as a batch separator', () => {
    assert.equal(
      unformatSql('SELECT 1 /*\nGO\n*/ + 1'),
      'SELECT 1 /* GO*/ + 1'
    );
  });

  test('preserves keyword case and collapses multiple statements', () => {
    assert.equal(
      unformatSql('select  *  from t;\nselect  1;'),
      'select * from t; select 1;'
    );
  });

  test('returns empty input unchanged', () => {
    assert.equal(unformatSql(''), '');
    assert.equal(unformatSql('   '), '   ');
  });
});
