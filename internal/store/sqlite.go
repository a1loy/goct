package store

import (
	"bytes"
	"database/sql"
	"fmt"
	"goct/internal/config"
	"goct/internal/models"
	"html/template"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultCreateQuery = `
  CREATE TABLE IF NOT EXISTS {{.TableName}} (
  id INTEGER NOT NULL PRIMARY KEY,
  name TEXT,
  entry TEXT,
  common_name TEXT,
  hash TEXT,
  raw TEXT
  );`
	insertQuery = "INSERT INTO {{.TableName}} VALUES(NULL,?,?,?,?,?)"
	storeType   = "sqlite"
)

type SqliteStoreClient struct {
	Name        string
	FilePath    string
	DB          *sql.DB
	TableName   string
	insertQuery string
	flush       bool
	ready       bool
}

func buildQuery(templateStr string, tableName string) (string, error) {
	tplStruct := struct {
		TableName string
	}{
		TableName: tableName,
	}
	var buf bytes.Buffer
	tpl, err := template.New("").Parse(templateStr)
	if err != nil {
		return "", err
	}
	err = tpl.Execute(&buf, tplStruct)
	return buf.String(), err
}

func NewSqliteStoreClient(cfg config.Config) (*SqliteStoreClient, error) {
	for _, storeCfg := range cfg.GetStoreConfigs() {
		if storeCfg.Type == storeType {
			createNewDB := storeCfg.Flush
			dbPath := strings.TrimPrefix(storeCfg.ConnString, "file:")
			insertQuery, err := buildQuery(insertQuery, storeCfg.TableName)
			if err != nil {
				return nil, err
			}
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				// database doesnt exists
				if createNewDB {
					// return &SqliteStoreClient{storeType, dbPath, nil, storeCfg.TableName, insertQuery, true, true}, nil
					return &SqliteStoreClient{Name: storeType, FilePath: dbPath, DB: nil,
						TableName: storeCfg.TableName, insertQuery: insertQuery, flush: true, ready: true}, nil
				}
			} else {
				// database exists
				return &SqliteStoreClient{Name: storeType, FilePath: dbPath, DB: nil,
					TableName: storeCfg.TableName, insertQuery: insertQuery, flush: createNewDB, ready: true}, nil
			}
		}
	}
	return &SqliteStoreClient{Name: "", FilePath: "", DB: nil,
		TableName: "", insertQuery: "", flush: true, ready: false}, fmt.Errorf("unable to find config for %s", storeType)
}

func (c *SqliteStoreClient) Init() error {
	if c.flush {
		werr := os.WriteFile(c.FilePath, []byte{}, 0644)
		if werr != nil {
			return werr
		}
	}
	db, err := sql.Open("sqlite3", c.FilePath)
	if err != nil {
		return err
	}
	c.DB = db
	createQuery, cerr := buildQuery(defaultCreateQuery, c.TableName)
	if cerr != nil {
		return cerr
	}
	if _, err := c.DB.Exec(createQuery); err != nil {
		return err
	}
	return nil
}

func (c *SqliteStoreClient) Store(msg models.DetectMsg) error {
	if c.DB == nil {
		panic("unable to find DB instance")
	}
	_, err := c.DB.Exec(c.insertQuery, msg.Name, msg.Entry, msg.CN, msg.Hash, msg.Raw)
	if err != nil {
		return err
	}
	return nil
}

func (c *SqliteStoreClient) IsReady() bool {
	return c.ready
}
