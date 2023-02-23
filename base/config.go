package base

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/os/gctx"
	"gopkg.in/yaml.v2"
	"hei-comic-api/app/collector/adapter"
)

// InitAdapterCfg 加载采集器配置到全局
func InitAdapterCfg() error {
	err := g.Cfg().GetAdapter().(*gcfg.AdapterFile).SetPath("./config")
	if err != nil {
		return err
	}
	return nil
}

func InitAdapterTable() error {
	ctx := gctx.New()
	adapterVar, err := g.Cfg().Get(ctx, "adapter")
	if err != nil {
		return err
	}
	baseApt := adapter.BaseAdapter{}
	adapterVars := adapterVar.Vars()
	for _, cfgVar := range adapterVars {
		adapter := adapter.Adapter{}
		err = yaml.Unmarshal(cfgVar.Bytes(), &adapter)
		if err != nil {
			return err
		}
		err = baseApt.InitTables(adapter.OrderId, adapter.Name)
		if err != nil {
			return err
		}

	}
	//baseApt.TongTable(6)
	return nil
}
