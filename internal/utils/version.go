package utils

import "runtime/debug"

var (
	azureTLSVersion = "unknown"
)

func init() {
	azureTLSVersion = "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/Noooste/azuretls-client" {
				azureTLSVersion = dep.Version
				break
			}
		}
	}
}

func GetAzureTLSVersion() string {
	return azureTLSVersion
}
