package lib

import "github.com/gocql/gocql"

var (
	Session *gocql.Session
	Cfg 	ConfigStruct
)

type Context struct {
	Session *gocql.Session
	Cfg *ConfigStruct
}

func (c *Context) Close() {
	c.Session.Close()
}

func GetContext() (*Context, error) {
	return &Context{
		Session: Session,
		Cfg: &Cfg,
	}, nil
}