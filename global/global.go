package global

import "syscall/js"

var DomRegistry = make(map[string]js.Value)
var IncrementCounter = 0
