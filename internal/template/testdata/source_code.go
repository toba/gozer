package gota

import (
	"fmt"
)

var atomicInt int = 23
var atomicString string = "hello"
var atomicBool bool = true

var varFunction = func(int, bool) string { return "" }
var varAnyFunction any = func(int, bool) string { return "" }
var varStructSimple Ast
var varStructWithMethods Person
var varSliceSimple []int
var varMapSimple map[string]*int
var varMapStruct MapStruct
var varPointerSimple *int
var varInterfaceError error
var varInterfaceEmbeded EmbededInterface
var varChannelSimple chan int
var varAdvancedData *AdvancedData

// Iterator types (Go 1.23+)
var varIterSeq func(yield func(int) bool)                    // iter.Seq[int]
var varIterSeq2 func(yield func(string, int) bool)           // iter.Seq2[string, int]
var varIterSeqPerson func(yield func(*Person) bool)          // iter.Seq[*Person]
var varIterSeq2StringAst func(yield func(string, Ast) bool)  // iter.Seq2[string, Ast]

type Ast struct {
	Kind int
	Data any
}

type DeathAst Ast
type CopeAst = Ast

func (c CopeAst) String() string {
	return fmt.Sprintln("type of CopeAst struct {}")
}

type Person struct {
	name    string
	Age     int
	message chan string
}

func (p Person) Name() string
func (p Person) GetAge() int
func (p *Person) SetAge(int)
func (p Person) GetAst() Ast

type textWriter interface {
	WriteText(string) int
	Err() error
}

type EmbededInterface interface {
	error
	textWriter

	ReadText() string
}

type MapStruct struct {
	table map[string]*int
}

type AdvancedData struct {
	person      *Person
	Counter     *int
	Channel     chan string
	EmbederFace EmbededInterface
}

func init() {
	varInterfaceError.Error()
}
