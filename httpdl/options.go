package httpdl

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

type about bool

func (a about) BeforeApply() (err error) {
	fmt.Println("Visit https://github.com/gonejack/httpdl")
	os.Exit(0)
	return
}

type Options struct {
	Username string `short:"u" name:"username" help:"http文件服务器用户名"`
	Password string `short:"p" name:"password" help:"http文件服务器用户名密码"`
	Download string `short:"d" name:"download-url" help:"http文件服务路径"`
	Verbose  bool   `short:"v" help:"Verbose"`
	About    about  `help:"About."`
}

func (o *Options) PrintUsage() {
	_ = kong.Parse(o).PrintUsage(true)
}

func MustParseOptions() (opts Options) {
	kong.Parse(&opts,
		kong.Name("httpdl"),
		kong.Description("http文件服务器递归下载"),
		kong.UsageOnError(),
	)
	return
}
