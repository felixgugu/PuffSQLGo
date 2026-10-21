export namespace model {
	
	export class ColumnChangeIssue {
	    code: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ColumnChangeIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	    }
	}
	export class ColumnDependency {
	    kind: string;
	    name: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new ColumnDependency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.detail = source["detail"];
	    }
	}
	export class ColumnChangePreview {
	    sql: string;
	    fingerprint: string;
	    warnings: ColumnChangeIssue[];
	    blockingIssues: ColumnChangeIssue[];
	    dependencies: ColumnDependency[];
	    affectedRows: number;
	
	    static createFrom(source: any = {}) {
	        return new ColumnChangePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sql = source["sql"];
	        this.fingerprint = source["fingerprint"];
	        this.warnings = this.convertValues(source["warnings"], ColumnChangeIssue);
	        this.blockingIssues = this.convertValues(source["blockingIssues"], ColumnChangeIssue);
	        this.dependencies = this.convertValues(source["dependencies"], ColumnDependency);
	        this.affectedRows = source["affectedRows"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ColumnChangeRequest {
	    connectionId: string;
	    database: string;
	    schema: string;
	    table: string;
	    operation: string;
	    originalName?: string;
	    columnName: string;
	    dataType: string;
	    length?: number;
	    precision?: number;
	    scale?: number;
	    isNullable: boolean;
	    defaultExpression?: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new ColumnChangeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connectionId = source["connectionId"];
	        this.database = source["database"];
	        this.schema = source["schema"];
	        this.table = source["table"];
	        this.operation = source["operation"];
	        this.originalName = source["originalName"];
	        this.columnName = source["columnName"];
	        this.dataType = source["dataType"];
	        this.length = source["length"];
	        this.precision = source["precision"];
	        this.scale = source["scale"];
	        this.isNullable = source["isNullable"];
	        this.defaultExpression = source["defaultExpression"];
	        this.description = source["description"];
	    }
	}
	export class ColumnChangeResult {
	    executionTimeMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ColumnChangeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.executionTimeMs = source["executionTimeMs"];
	    }
	}
	export class ColumnDef {
	    name: string;
	    dataType: string;
	    nullable: boolean;
	    ordinal: number;
	
	    static createFrom(source: any = {}) {
	        return new ColumnDef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.dataType = source["dataType"];
	        this.nullable = source["nullable"];
	        this.ordinal = source["ordinal"];
	    }
	}
	
	export class ColumnItem {
	    name: string;
	    dataType: string;
	    maxLength?: number;
	    precision?: number;
	    scale?: number;
	    isNullable: boolean;
	    isPrimaryKey: boolean;
	    isIdentity: boolean;
	    isComputed: boolean;
	    isRowVersion: boolean;
	    defaultDefinition?: string;
	    collation?: string;
	    computedExpression?: string;
	    isPersisted: boolean;
	    identitySeed?: number;
	    identityIncrement?: number;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new ColumnItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.dataType = source["dataType"];
	        this.maxLength = source["maxLength"];
	        this.precision = source["precision"];
	        this.scale = source["scale"];
	        this.isNullable = source["isNullable"];
	        this.isPrimaryKey = source["isPrimaryKey"];
	        this.isIdentity = source["isIdentity"];
	        this.isComputed = source["isComputed"];
	        this.isRowVersion = source["isRowVersion"];
	        this.defaultDefinition = source["defaultDefinition"];
	        this.collation = source["collation"];
	        this.computedExpression = source["computedExpression"];
	        this.isPersisted = source["isPersisted"];
	        this.identitySeed = source["identitySeed"];
	        this.identityIncrement = source["identityIncrement"];
	        this.description = source["description"];
	    }
	}
	export class ConnectionProfile {
	    name: string;
	    alias?: string;
	    host: string;
	    port: number;
	    database: string;
	    username: string;
	    encrypt: boolean;
	    trustServerCertificate: boolean;
	    color?: string;
	    modificationPrompt?: boolean;
	    id: string;
	    engine: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.alias = source["alias"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.database = source["database"];
	        this.username = source["username"];
	        this.encrypt = source["encrypt"];
	        this.trustServerCertificate = source["trustServerCertificate"];
	        this.color = source["color"];
	        this.modificationPrompt = source["modificationPrompt"];
	        this.id = source["id"];
	        this.engine = source["engine"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class CustomTemplateItem {
	    id: string;
	    title: string;
	    category: string;
	    categoryLabel?: string;
	    tags: string[];
	    description: string;
	    code: string;
	    isCustom?: boolean;
	    createdAt?: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomTemplateItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.category = source["category"];
	        this.categoryLabel = source["categoryLabel"];
	        this.tags = source["tags"];
	        this.description = source["description"];
	        this.code = source["code"];
	        this.isCustom = source["isCustom"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class CustomTemplatesResponse {
	    filePath: string;
	    templates: CustomTemplateItem[];
	
	    static createFrom(source: any = {}) {
	        return new CustomTemplatesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.templates = this.convertValues(source["templates"], CustomTemplateItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DatabaseItem {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	export class ExcelExportResult {
	    canceled: boolean;
	    filePath: string;
	    tableCount: number;
	    sheetCount: number;
	    columnCount: number;
	    totalRows: number;
	    truncatedCount: number;
	    executionTimeMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ExcelExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canceled = source["canceled"];
	        this.filePath = source["filePath"];
	        this.tableCount = source["tableCount"];
	        this.sheetCount = source["sheetCount"];
	        this.columnCount = source["columnCount"];
	        this.totalRows = source["totalRows"];
	        this.truncatedCount = source["truncatedCount"];
	        this.executionTimeMs = source["executionTimeMs"];
	    }
	}
	export class ExcelTableRef {
	    schema: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new ExcelTableRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema = source["schema"];
	        this.name = source["name"];
	    }
	}
	export class ForeignKeyItem {
	    constraintName: string;
	    fromSchema: string;
	    fromTable: string;
	    fromColumn: string;
	    toSchema: string;
	    toTable: string;
	    toColumn: string;
	
	    static createFrom(source: any = {}) {
	        return new ForeignKeyItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.constraintName = source["constraintName"];
	        this.fromSchema = source["fromSchema"];
	        this.fromTable = source["fromTable"];
	        this.fromColumn = source["fromColumn"];
	        this.toSchema = source["toSchema"];
	        this.toTable = source["toTable"];
	        this.toColumn = source["toColumn"];
	    }
	}
	export class ImportCapabilities {
	    identityColumn?: string;
	    writableColumnCount: number;
	    canAlterTable: boolean;
	    engineEdition: number;
	    supportsIdentityInsert: boolean;
	    disabledReason?: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.identityColumn = source["identityColumn"];
	        this.writableColumnCount = source["writableColumnCount"];
	        this.canAlterTable = source["canAlterTable"];
	        this.engineEdition = source["engineEdition"];
	        this.supportsIdentityInsert = source["supportsIdentityInsert"];
	        this.disabledReason = source["disabledReason"];
	    }
	}
	export class ImportRowError {
	    line: number;
	    column?: string;
	    message: string;
	    serverCode?: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportRowError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.column = source["column"];
	        this.message = source["message"];
	        this.serverCode = source["serverCode"];
	    }
	}
	export class ImportResult {
	    insertedCount: number;
	    rolledBack: boolean;
	    errors: ImportRowError[];
	    executionTimeMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.insertedCount = source["insertedCount"];
	        this.rolledBack = source["rolledBack"];
	        this.errors = this.convertValues(source["errors"], ImportRowError);
	        this.executionTimeMs = source["executionTimeMs"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ImportRowPayload {
	    line: number;
	    values: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportRowPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.values = source["values"];
	    }
	}
	export class IndexColumnRef {
	    name: string;
	    descending: boolean;
	    isIncluded: boolean;
	
	    static createFrom(source: any = {}) {
	        return new IndexColumnRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.descending = source["descending"];
	        this.isIncluded = source["isIncluded"];
	    }
	}
	export class KeyColumnRef {
	    name: string;
	    descending: boolean;
	
	    static createFrom(source: any = {}) {
	        return new KeyColumnRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.descending = source["descending"];
	    }
	}
	export class QueryMessage {
	    level: string;
	    message: string;
	    code?: number;
	    lineNumber?: number;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new QueryMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.message = source["message"];
	        this.code = source["code"];
	        this.lineNumber = source["lineNumber"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class ResultSet {
	    columns: ColumnDef[];
	    rows: any[][];
	    rowCount: number;
	    totalCount: number;
	    isTruncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResultSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = this.convertValues(source["columns"], ColumnDef);
	        this.rows = source["rows"];
	        this.rowCount = source["rowCount"];
	        this.totalCount = source["totalCount"];
	        this.isTruncated = source["isTruncated"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class QueryResult {
	    resultSets: ResultSet[];
	    messages: QueryMessage[];
	    affectedRows: number;
	    executionTimeMs: number;
	
	    static createFrom(source: any = {}) {
	        return new QueryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resultSets = this.convertValues(source["resultSets"], ResultSet);
	        this.messages = this.convertValues(source["messages"], QueryMessage);
	        this.affectedRows = source["affectedRows"];
	        this.executionTimeMs = source["executionTimeMs"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SQLFileNode {
	    name: string;
	    path: string;
	    is_dir: boolean;
	    size?: number;
	    modified_time?: string;
	    children?: SQLFileNode[];
	
	    static createFrom(source: any = {}) {
	        return new SQLFileNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.is_dir = source["is_dir"];
	        this.size = source["size"];
	        this.modified_time = source["modified_time"];
	        this.children = this.convertValues(source["children"], SQLFileNode);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SaveConnectionRequest {
	    name: string;
	    alias?: string;
	    host: string;
	    port: number;
	    database: string;
	    username: string;
	    encrypt: boolean;
	    trustServerCertificate: boolean;
	    color?: string;
	    modificationPrompt?: boolean;
	    id?: string;
	    password?: string;
	    copyPasswordFrom?: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveConnectionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.alias = source["alias"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.database = source["database"];
	        this.username = source["username"];
	        this.encrypt = source["encrypt"];
	        this.trustServerCertificate = source["trustServerCertificate"];
	        this.color = source["color"];
	        this.modificationPrompt = source["modificationPrompt"];
	        this.id = source["id"];
	        this.password = source["password"];
	        this.copyPasswordFrom = source["copyPasswordFrom"];
	    }
	}
	export class TableConstraintItem {
	    name: string;
	    kind: string;
	    keyColumns: KeyColumnRef[];
	    definition?: string;
	    defaultColumn?: string;
	    isClustered?: boolean;
	    isDisabled: boolean;
	    isNotTrusted: boolean;
	    refSchema?: string;
	    refTable?: string;
	    refColumns: string[];
	    onDelete?: string;
	    onUpdate?: string;
	
	    static createFrom(source: any = {}) {
	        return new TableConstraintItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.keyColumns = this.convertValues(source["keyColumns"], KeyColumnRef);
	        this.definition = source["definition"];
	        this.defaultColumn = source["defaultColumn"];
	        this.isClustered = source["isClustered"];
	        this.isDisabled = source["isDisabled"];
	        this.isNotTrusted = source["isNotTrusted"];
	        this.refSchema = source["refSchema"];
	        this.refTable = source["refTable"];
	        this.refColumns = source["refColumns"];
	        this.onDelete = source["onDelete"];
	        this.onUpdate = source["onUpdate"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TableIndexItem {
	    name: string;
	    typeDesc: string;
	    isUnique: boolean;
	    isPrimaryKey: boolean;
	    isUniqueConstraint: boolean;
	    isDisabled: boolean;
	    columns: IndexColumnRef[];
	    filterDefinition?: string;
	
	    static createFrom(source: any = {}) {
	        return new TableIndexItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.typeDesc = source["typeDesc"];
	        this.isUnique = source["isUnique"];
	        this.isPrimaryKey = source["isPrimaryKey"];
	        this.isUniqueConstraint = source["isUniqueConstraint"];
	        this.isDisabled = source["isDisabled"];
	        this.columns = this.convertValues(source["columns"], IndexColumnRef);
	        this.filterDefinition = source["filterDefinition"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TableItem {
	    schema: string;
	    name: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new TableItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema = source["schema"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	    }
	}
	export class TableTriggerItem {
	    name: string;
	    isInsteadOf: boolean;
	    isDisabled: boolean;
	    events: string[];
	    definition?: string;
	
	    static createFrom(source: any = {}) {
	        return new TableTriggerItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isInsteadOf = source["isInsteadOf"];
	        this.isDisabled = source["isDisabled"];
	        this.events = source["events"];
	        this.definition = source["definition"];
	    }
	}
	export class TableObjects {
	    columns: ColumnItem[];
	    constraints: TableConstraintItem[];
	    indexes: TableIndexItem[];
	    triggers: TableTriggerItem[];
	
	    static createFrom(source: any = {}) {
	        return new TableObjects(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = this.convertValues(source["columns"], ColumnItem);
	        this.constraints = this.convertValues(source["constraints"], TableConstraintItem);
	        this.indexes = this.convertValues(source["indexes"], TableIndexItem);
	        this.triggers = this.convertValues(source["triggers"], TableTriggerItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TableSchema {
	    schema: string;
	    name: string;
	    kind: string;
	    columns: ColumnItem[];
	
	    static createFrom(source: any = {}) {
	        return new TableSchema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema = source["schema"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.columns = this.convertValues(source["columns"], ColumnItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

