// For https://github.com/jo3-l/action-check-yag-tmpl-syntax
package genfuncdata

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/jonas747/dcmd/v3"
	"github.com/jonas747/discordgo"
	"github.com/jonas747/yagpdb/commands"
	"github.com/jonas747/yagpdb/common/templates"
	"github.com/jonas747/yagpdb/stdcommands/util"
)

type FuncData struct {
	Name     string `json:"name"`
	NumIn    int    `json:"num_in"`
	Variadic bool   `json:"variadic"`
}

func newFuncData(f interface{}, name string) *FuncData {
	typ := reflect.ValueOf(f).Elem().Type()
	return &FuncData{Name: name, NumIn: typ.NumIn(), Variadic: typ.IsVariadic()}
}

var Command = &commands.YAGCommand{
	CmdCategory:          commands.CategoryDebug,
	HideFromCommandsPage: true,
	Name:                 "genfuncdata",
	Description:          ":O",
	HideFromHelp:         true,
	RunFunc: util.RequireOwner(func(data *dcmd.Data) (interface{}, error) {
		ctx := templates.NewContext(data.GuildData.GS, data.GuildData.CS, data.GuildData.MS)
		// ctx.ContextFuncs is lazily loaded, so call Parse() once with a dummy input to make it show up.
		ctx.Parse("")
		funcData := make([]*FuncData, 0, len(templates.StandardFuncMap)+len(ctx.ContextFuncs))
		for name, fun := range templates.StandardFuncMap {
			funcData = append(funcData, newFuncData(fun, name))
		}
		for name, fun := range ctx.ContextFuncs {
			funcData = append(funcData, newFuncData(fun, name))
		}

		res, err := json.Marshal(funcData)
		if err != nil {
			return "Failed marshalling data", err
		}

		buf := bytes.NewBuffer(res)
		file := &discordgo.File{ContentType: "application/json", Name: "funcs.json", Reader: buf}
		return &discordgo.MessageSend{Files: []*discordgo.File{file}}, nil
	}),
}
