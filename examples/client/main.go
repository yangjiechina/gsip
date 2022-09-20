package main

import (
	"fmt"
	"gsip/examples"
	"net/http"
	_ "net/http"
	_ "net/http/pprof"
)

func main() {
	if config, err := examples.ReadConfig("config.json"); err != nil {
		fmt.Printf("读取配置文件失败:%s", err.Error())
	} else {
		go func() {
			loadConfigError := http.ListenAndServe(":19999", nil)
			if loadConfigError != nil {
				panic(loadConfigError)
			}
			println("浏览器打开GO调试页面:http://localhost:19999/debug/pprof/")
		}()

		StartClient(config.Client)
	}

	for {
		select {}
	}
}
