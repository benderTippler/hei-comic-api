package snowflake

import (
	"github.com/bwmarrin/snowflake"
)

var snowNode *snowflake.Node

func InitSnowflake() error {
	var err error
	snowNode, err = snowflake.NewNode(1)
	if err != nil {
		return err
	}
	return nil
}

func NewSnowNode() *snowflake.Node {
	return snowNode
}
