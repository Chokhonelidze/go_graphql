data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./graph/model",
    "--dialect", "sqlserver",
  ]
}

env "compose" {
  src = data.external_schema.gorm.url
  dev = getenv("DB_CONNECTION_STRING")

  migration {
    dir    = "file://migrations/sql"
    format = golang-migrate
  }
}