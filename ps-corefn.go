package main

type psCoreFn struct {
	ModuleName []string    `json:"moduleName"`
	ModulePath string      `json:"modulePath"`
	Imports    interface{} `json:"imports"`
}
