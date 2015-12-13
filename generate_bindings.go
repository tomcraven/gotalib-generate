package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

/*
#cgo LDFLAGS: -L/home/tom/dev/ta-lib/src/.libs/ -lta_lib -lm
#include "/home/tom/dev/ta-lib/include/ta_abstract.h"
char* getListAt(char **list, unsigned int idx)
{
	return list[idx];
}
TA_IntegerDataPair getIntegerDataPairAt( TA_IntegerDataPair* data, unsigned int index )
{
	return data[index];
}
*/
import "C"

func getGroups() []string {
	var table *C.TA_StringTable
	retCode := C.TA_GroupTableAlloc(&table)

	if retCode == C.TA_SUCCESS {
		defer C.TA_FuncTableFree(table)
		ret := make([]string, table.size)
		for i := C.uint(0); i < table.size; i++ {
			ret[i] = C.GoString(C.getListAt(table.string, i))
		}
		return ret
	}

	return []string{}
}

func getFunctions(group string) []string {
	var table *C.TA_StringTable
	retCode := C.TA_FuncTableAlloc(C.CString(group), &table)

	if retCode == C.TA_SUCCESS {
		defer C.TA_FuncTableFree(table)
		ret := make([]string, table.size)
		for i := C.uint(0); i < table.size; i++ {
			ret[i] = C.GoString(C.getListAt(table.string, i))
		}
		return ret
	}

	return []string{}
}

func getFunctionHandle(functionName string) *C.TA_FuncHandle {
	var handle *C.TA_FuncHandle
	C.TA_GetFuncHandle(C.CString(functionName), &handle)
	return handle
}

func getFunctionInfo(handle *C.TA_FuncHandle) C.TA_FuncInfo {
	var info *C.TA_FuncInfo
	C.TA_GetFuncInfo(handle, &info)
	return *info
}

func writeToBindingsFile(data string) {
	if _, err := bindingsOutputFile.WriteString(data); err != nil {
		panic(err)
	}
}

func initBindings() {
	writeToBindingsFile(
		"package " + kLibraryName + "\n\n" +
			"import \"math\"\n\n" +
			"/*\n" +
			"#cgo LDFLAGS: -L/home/tom/dev/ta-lib/src/.libs/ -lta_lib -lm\n" +
			"#include \"/home/tom/dev/ta-lib/include/ta_abstract.h\"\n" +

			"*/\n" +
			"import \"C\"\n\n",
	)
}

func addStruct(name string) {
	writeToBindingsFile(
		"type " + name + " struct {\n" +
			"\tparams *C.TA_ParamHolder\n" +
			"\thandle *C.TA_FuncHandle\n\n" +

			"\trealInputByIndex map[int][]C.TA_Real\n" +
			"\tintegerOutputByIndex map[int][]C.TA_Integer\n" +
			"\trealOutputByIndex map[int][]C.TA_Real\n\n" +

			"\tfiddleValues []float64\n" +
			"}\n\n",
	)
}

func addInitFunction(name string, info C.TA_FuncInfo) {
	writeToBindingsFile(
		"func ( a *" + name + " ) init() {\n" +
			"\thelper_paramHolderAlloc( a.handle, &a.params )\n" +
			"\ta.realInputByIndex = make( map[int][]C.TA_Real )\n" +
			"\ta.realOutputByIndex = make( map[int][]C.TA_Real )\n" +
			"\ta.integerOutputByIndex = make( map[int][]C.TA_Integer )\n" +
			"\ta.fiddleValues = make( []float64, " + strconv.Itoa(int(info.nbOptInput)) + " )\n\n",
	)

	for i := 0; i < int(info.nbOptInput); i++ {
		var paramInfo *C.TA_OptInputParameterInfo
		C.TA_GetOptInputParameterInfo(info.handle, C.uint(i), &paramInfo)

		writeToBindingsFile(
			"\ta.fiddleValues[" + strconv.Itoa(i) + "] = float64( " +
				strconv.FormatFloat(float64(paramInfo.defaultValue), 'f', 10, 64) +
				" )\n",
		)
	}

	writeToBindingsFile("}\n\n")
}

func addGetNumInputsFunction(name string, info C.TA_FuncInfo) {
	writeToBindingsFile(
		"func ( a *" + name + " ) GetNumInputs() ( int ) {\n" +
			"\treturn " + strconv.Itoa(int(info.nbInput)) + "\n" +
			"}\n\n",
	)
}

func addSetInputDataFunction(name string, info C.TA_FuncInfo) {
	writeToBindingsFile(
		"func ( a *" + name + " ) SetInputData( index int, data []float64 ) {\n",
	)

	for i := 0; i < int(info.nbInput); i++ {
		var paramInfo *C.TA_InputParameterInfo
		C.TA_GetInputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_Input_Real {
			writeToBindingsFile(
				"\tif index == " + strconv.Itoa(i) + " {\n" +
					"\t\tvar temp []C.TA_Real\n" +
					"\t\thelper_convertGoFloat64ArrayToTaRealArray( data, &temp )\n" +
					"\t\ta.realInputByIndex[index] = temp\n" +
					"\t\thelper_setInputDataReal( a.params, index, &(a.realInputByIndex[index][0]) )\n" +
					"\t\treturn\n" +
					"\t}\n",
			)
		} else if paramInfo._type == C.TA_Input_Price {
			// Do nothing, handled later...
		} else {
			panic("doh 1")
		}

	}

	writeToBindingsFile(
		"}\n\n",
	)
}

func addSetPriceInputDataFunction(name string, info C.TA_FuncInfo) {
	writeToBindingsFile(
		"func ( a *" + name + " ) SetPriceInputData( open, high, low, close, volume, openInterest []float64 ) {\n",
	)

	for i := 0; i < int(info.nbInput); i++ {
		var paramInfo *C.TA_InputParameterInfo
		C.TA_GetInputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_Input_Price {
			writeToBindingsFile(
				"\tvar temp []C.TA_Real\n" +
					"\thelper_convertGoFloat64ArrayToTaRealArray( open, &temp )\n" +
					"\ta.realInputByIndex[0] = temp\n" +
					"\thelper_convertGoFloat64ArrayToTaRealArray( high, &temp )\n" +
					"\ta.realInputByIndex[1] = temp\n" +
					"\thelper_convertGoFloat64ArrayToTaRealArray( low, &temp )\n" +
					"\ta.realInputByIndex[2] = temp\n" +
					"\thelper_convertGoFloat64ArrayToTaRealArray( close, &temp )\n" +
					"\ta.realInputByIndex[3] = temp\n" +
					"\thelper_convertGoFloat64ArrayToTaRealArray( volume, &temp )\n" +
					"\ta.realInputByIndex[4] = temp\n" +
					"\thelper_convertGoFloat64ArrayToTaRealArray( openInterest, &temp )\n" +
					"\ta.realInputByIndex[5] = temp\n" +
					"\thelper_setInputDataPrice( \n" +
					"\t\ta.params, " + strconv.Itoa(i) + ", \n" +
					"\t\t&( a.realInputByIndex[0][0] ), \n" +
					"\t\t&( a.realInputByIndex[1][0] ), \n" +
					"\t\t&( a.realInputByIndex[2][0] ), \n" +
					"\t\t&( a.realInputByIndex[3][0] ), \n" +
					"\t\t&( a.realInputByIndex[4][0] ), \n" +
					"\t\t&( a.realInputByIndex[5][0] ),\n" +
					"\t)\n",
			)
			break
		}
	}

	writeToBindingsFile(
		"}\n\n",
	)
}

func addGetAndSetFiddleValuesFunctions(name string, info C.TA_FuncInfo) {
	writeToBindingsFile(
		"func ( a *" + name + " ) GetNumFiddleValues() int {\n" +
			"\treturn " + strconv.Itoa(int(info.nbOptInput)) + "\n" +
			"}\n" +
			"\n" +
			"func ( a *" + name + " ) GetFiddleValues() ( []float64 ) {\n" +
			"\treturn a.fiddleValues\n" +
			"}\n" +
			"\n" +
			"func ( a *" + name + " ) SetFiddleValues( v []float64 ) {\n" +
			"\tif len( v ) != " + strconv.Itoa(int(info.nbOptInput)) + " {\n" +
			"\t	panic( \"SetFiddleValues : bad number of fiddle values passed\" )\n" +
			"\t}\n" +
			"\ta.fiddleValues = v\n" +
			"}\n\n",
	)
}

func addGetNumOutputValues(name string, info C.TA_FuncInfo) {
	writeToBindingsFile(
		"func (a *" + name + " ) GetNumOutputValues() ( int ) {\n" +
			"\treturn " + strconv.Itoa(int(info.nbOutput)) + "\n" +
			"}\n\n",
	)
}

func addFixFiddleValueFunction(name string, info C.TA_FuncInfo) {
	writeToBindingsFile(
		"func (a *" + name + " ) FixFiddleValue( fiddleValueIndex int, inValue float64 ) ( float64 ) {\n",
	)

	for i := 0; i < int(info.nbOptInput); i++ {
		var paramInfo *C.TA_OptInputParameterInfo
		C.TA_GetOptInputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_OptInput_IntegerRange {
			integerRange := (*C.TA_IntegerRange)(unsafe.Pointer(paramInfo.dataSet))

			writeToBindingsFile(
				"\tif fiddleValueIndex == " + strconv.Itoa(i) + " {\n" +
					fmt.Sprintf("\t\treturn float64( int( %d + ( float64( %d - %d ) * inValue ) ) )\n",
						int(integerRange.suggested_start),
						int(integerRange.suggested_end),
						int(integerRange.suggested_start),
					) +
					"\t}\n\n",
			)
		} else if paramInfo._type == C.TA_OptInput_RealRange {
			realRange := (*C.TA_RealRange)(unsafe.Pointer(paramInfo.dataSet))

			writeToBindingsFile(
				"\tif fiddleValueIndex == " + strconv.Itoa(i) + " {\n" +
					fmt.Sprintf("\t\treturn %f + ( ( %f - %f ) * inValue )\n",
						float64(realRange.suggested_start),
						float64(realRange.suggested_end),
						float64(realRange.suggested_start),
					) +
					"\t}\n\n",
			)
		} else if paramInfo._type == C.TA_OptInput_IntegerList {
			integerList := (*C.TA_IntegerList)(unsafe.Pointer(paramInfo.dataSet))

			writeToBindingsFile(
				"\tif fiddleValueIndex == " + strconv.Itoa(i) + " {\n" +
					fmt.Sprintf("\t\tindex := int( inValue * float64( %d ) )\n", int(integerList.nbElement)) +
					"\t\tindexToRetMap := map[int]int {\n",
			)

			for i := 0; i < int(integerList.nbElement); i++ {

				writeToBindingsFile(
					"\t\t\t" + strconv.Itoa(i) + " : " +
						strconv.Itoa(int(C.getIntegerDataPairAt(integerList.data, C.uint(i)).value)) + ",\n",
				)
			}

			writeToBindingsFile(
				"\t\t}\n" +
					"\t\treturn float64( indexToRetMap[index] )\n" +
					"\t}\n\n",
			)

		} else {
			fmt.Println(paramInfo._type)
			panic("doh 7")
		}
	}

	writeToBindingsFile("\tpanic(\"invalid index passed to FixFiddleValue\")\n")
	writeToBindingsFile("\treturn 0.0\n")
	writeToBindingsFile("}\n\n")
}

func addGoFunction(name string, info C.TA_FuncInfo) {

	writeToBindingsFile(
		"func ( a *" + name + " ) Go( outIndex int ) ( []float64 ) {\n" +
			"\tstartIndex := 0\n",
	)

	writeToBindingsFile(
		"\tfor i := 1; i < " + strconv.Itoa(int(info.nbInput)) + "; i++ {\n" +
			"\t\tif len( a.realInputByIndex[i] ) != len( a.realInputByIndex[i - 1] ) {\n" +
			"\t\t\tpanic(\"Input data has different lengths\")\n" +
			"\t\t}\n" +
			"\t}\n",
	)

	writeToBindingsFile(
		"\tendIndex := len( a.realInputByIndex[0] ) - 1\n\n",
	)

	for i := 0; i < int(info.nbOptInput); i++ {
		var paramInfo *C.TA_OptInputParameterInfo
		C.TA_GetOptInputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_OptInput_IntegerRange {
			writeToBindingsFile(
				"\thelper_setOptInputDataInteger( a.params, " + strconv.Itoa(i) + ", int( a.fiddleValues[" + strconv.Itoa(i) + "] ) )\n",
			)
		} else if paramInfo._type == C.TA_OptInput_RealRange {
			writeToBindingsFile(
				"\thelper_setOptInputDataReal( a.params, " + strconv.Itoa(i) + ", a.fiddleValues[" + strconv.Itoa(i) + "] )\n",
			)
		} else if paramInfo._type == C.TA_OptInput_IntegerList {
			writeToBindingsFile(
				"\thelper_setOptInputDataInteger( a.params, " + strconv.Itoa(i) + ", int( a.fiddleValues[" + strconv.Itoa(i) + "] ) )\n",
			)
		} else {
			panic("doh 6")
		}
	}

	writeToBindingsFile("\n")

	for i := 0; i < int(info.nbOutput); i++ {
		var paramInfo *C.TA_OutputParameterInfo
		C.TA_GetOutputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_Output_Real {
			writeToBindingsFile(
				"\ta.realOutputByIndex[" + strconv.Itoa(i) + "] = make( []C.TA_Real, endIndex - startIndex + 1 )\n" +
					"\thelper_setOutputParamRealPtr( a.params, " + strconv.Itoa(i) + ", &(a.realOutputByIndex[" + strconv.Itoa(i) + "][0] ) )\n",
			)
		} else if paramInfo._type == C.TA_Output_Integer {
			writeToBindingsFile(
				"\ta.integerOutputByIndex[" + strconv.Itoa(i) + "] = make( []C.TA_Integer, endIndex - startIndex + 1 )\n" +
					"\thelper_setOutputParamIntegerPtr( a.params, " + strconv.Itoa(i) + ", &(a.integerOutputByIndex[" + strconv.Itoa(i) + "][0] ) )\n",
			)
		} else {
			panic("doh 3")
		}
	}

	writeToBindingsFile("\n")

	writeToBindingsFile(
		"\tvar temp, numElements C.TA_Integer\n" +
			"\thelper_callFunction( a.params, startIndex, endIndex, &temp, &numElements )\n\n",
	)

	for i := 0; i < int(info.nbOutput); i++ {
		var paramInfo *C.TA_OutputParameterInfo
		C.TA_GetOutputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_Output_Real {
			writeToBindingsFile(
				"\tif outIndex == " + strconv.Itoa(i) + " {\n" +
					"\t\tvar ret []float64\n" +
					"\t\thelper_convertTaRealArrayToGoFloat64Array( a.realOutputByIndex[outIndex], &ret )\n" +
					"\t\treturn ret[:numElements]\n" +
					"\t}\n",
			)
		} else if paramInfo._type == C.TA_Output_Integer {
			writeToBindingsFile(
				"\tif outIndex == " + strconv.Itoa(i) + " {\n" +
					"\t\tvar ret []float64\n" +
					"\t\thelper_convertTaIntegerArrayToGoFloat64Array( a.integerOutputByIndex[outIndex], &ret )\n" +
					"\t\treturn ret[:numElements]\n" +
					"\t}\n",
			)
		} else {
			panic("doh 4")
		}
	}

	writeToBindingsFile("\n")

	// Add default return
	for i := 0; i < int(info.nbOutput); i++ {
		var paramInfo *C.TA_OutputParameterInfo
		C.TA_GetOutputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_Output_Real {
			writeToBindingsFile(
				"\tvar ret []float64\n" +
					"\thelper_convertTaRealArrayToGoFloat64Array( a.realOutputByIndex[" + strconv.Itoa(i) + "], &ret )\n" +
					"\treturn ret[:numElements]\n",
			)
			break
		} else if paramInfo._type == C.TA_Output_Integer {
			writeToBindingsFile(
				"\tvar ret []float64\n" +
					"\thelper_convertTaIntegerArrayToGoFloat64Array( a.integerOutputByIndex[" + strconv.Itoa(i) + "], &ret )\n" +
					"\treturn ret[:numElements]\n",
			)
			break
		} else {
			panic("doh 5")
		}
	}

	writeToBindingsFile("}\n\n")
}

func addGoSingleFunction(name string, info C.TA_FuncInfo) {

	writeToBindingsFile(
		"func ( a *" + name + " ) GoSingle( outputIndex int ) ( float64 ) {\n",
	)

	if shouldBeInTimePeriodArray(info) {

		timePeriodIndexes := []int{}
		for i := 0; i < int(info.nbOptInput); i++ {
			var paramInfo *C.TA_OptInputParameterInfo
			C.TA_GetOptInputParameterInfo(info.handle, C.uint(i), &paramInfo)

			if strings.Contains(C.GoString(paramInfo.displayName), "Period") {
				timePeriodIndexes = append(timePeriodIndexes, i)
			}
		}

		if len(timePeriodIndexes) > 0 {
			for _, timePeriodIndex := range timePeriodIndexes {
				writeToBindingsFile(
					"\ta.fiddleValues[" + strconv.Itoa(timePeriodIndex) + "] = float64( len( a.realInputByIndex[0] ) )\n",
				)
			}

			writeToBindingsFile(
				"\tret := a.Go( outputIndex )\n" +
					"\tif len( ret ) > 0 {\n" +
					"\t\tret := ret[0]\n" +
					"\t\tif math.IsNaN( ret ) || math.IsInf( ret, 0 ) {\n" +
					"\t\t\treturn 0\n" +
					"\t\t} else {\n" +
					"\t\t\treturn ret\n" +
					"\t\t}\n" +
					"\t} else {\n" +
					"\t\treturn 0\n" +
					"\t}\n",
			)
		} else {
			writeToBindingsFile(
				"\treturn 0\n",
			)
		}
	} else {
		writeToBindingsFile(
			"\treturn 0\n",
		)
	}

	writeToBindingsFile(
		"}\n\n",
	)
}

func addPubicCreateFunction(goFuncName, name string, info C.TA_FuncInfo) {
	writeToBindingsFile(
		"func " + goFuncName + "() TA_Function {\n" +
			"\tvar ret " + name + "\n" +
			"\thelper_getFunctionHandle( \"" + C.GoString(info.name) + "\", &ret.handle )\n" +
			"\tret.init()\n" +
			"\treturn &ret\n" +
			"}\n\n",
	)
}

func addSeparator() {
	writeToBindingsFile(
		"/*--------------------------------------------------------------------------------------------------------*/\n\n",
	)
}

func infoContainsRealInput(info C.TA_FuncInfo) bool {
	for i := 0; i < int(info.nbInput); i++ {
		var paramInfo *C.TA_InputParameterInfo
		C.TA_GetInputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_Input_Real {
			return true
		}
	}
	return false
}

func shouldBeInFunctionArray(info C.TA_FuncInfo) bool {
	for i := 0; i < int(info.nbInput); i++ {
		var paramInfo *C.TA_InputParameterInfo
		C.TA_GetInputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_Input_Price {
			if !infoContainsRealInput(info) {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

func createBinding(info C.TA_FuncInfo) {

	if !shouldBeInFunctionArray(info) && !shouldBeInTimePeriodArray(info) {
		return
	}

	camelCaseName := C.GoString(info.camelCaseName)
	structName := strings.ToLower(string(camelCaseName[0])) + camelCaseName[1:] + "_struct"

	addStruct(structName)
	addInitFunction(structName, info)
	addGetNumInputsFunction(structName, info)
	addSetInputDataFunction(structName, info)
	addSetPriceInputDataFunction(structName, info)
	addGetAndSetFiddleValuesFunctions(structName, info)
	addFixFiddleValueFunction(structName, info)
	addGetNumOutputValues(structName, info)
	addGoFunction(structName, info)
	addGoSingleFunction(structName, info)
	addPubicCreateFunction(camelCaseName, structName, info)
	addSeparator()
}

func writeFunctionArrayFile(data string) {
	if _, err := functionArrayOutputFile.WriteString(data); err != nil {
		panic(err)
	}
}

func initFunctionArray() {
	writeFunctionArrayFile(
		"package " + kLibraryName + "\n\n" +
			"var FunctionArray = []func()(TA_Function) {\n",
	)
}

func createFunctionArray(info C.TA_FuncInfo) {
	if shouldBeInFunctionArray(info) {
		writeFunctionArrayFile(
			"\t" + C.GoString(info.camelCaseName) + ",\n",
		)
	}
}

func shutdownFunctionArray() {
	writeFunctionArrayFile(
		"}\n",
	)
}

func writeTimePeriodArrayFile(data string) {
	if _, err := timePeriodArrayOutputFile.WriteString(data); err != nil {
		panic(err)
	}
}

func initTimePeriodArray() {
	writeTimePeriodArrayFile(
		"package " + kLibraryName + "\n\n" +

			"var TimePeriodFunctionArray = []func()(TA_Function) {\n",
	)
}

func shouldBeInTimePeriodArray(info C.TA_FuncInfo) bool {
	for i := 0; i < int(info.nbInput); i++ {
		var paramInfo *C.TA_InputParameterInfo
		C.TA_GetInputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if paramInfo._type == C.TA_Input_Real {
			return true
		}
	}
	return false
}

func createTimePeriodArray(info C.TA_FuncInfo) {
	if !shouldBeInTimePeriodArray(info) {
		return
	}

	timePeriodIndexes := []int{}
	for i := 0; i < int(info.nbOptInput); i++ {
		var paramInfo *C.TA_OptInputParameterInfo
		C.TA_GetOptInputParameterInfo(info.handle, C.uint(i), &paramInfo)

		if strings.Contains(C.GoString(paramInfo.displayName), "Period") {
			timePeriodIndexes = append(timePeriodIndexes, i)
		}
	}

	if len(timePeriodIndexes) > 0 {
		writeTimePeriodArrayFile(
			"\t" + C.GoString(info.camelCaseName) + ",\n",
		)
	}
}

func shutdownTimePeriodArray() {
	writeTimePeriodArrayFile(
		"}\n",
	)
}

func createTaFunctionFile() {
	if _, err := taFunctionOutputFile.WriteString(
		"package " + kLibraryName + "\n\n" +
			"type TA_Function interface {\n" +
			"\tinit()\n\n" +
			"\tGetNumInputs() ( int )\n" +
			"\tSetInputData( int, []float64 )\n" +
			"\tSetPriceInputData( []float64, []float64, []float64, []float64, []float64, []float64 )\n\n" +
			"\tGetNumFiddleValues() ( int )\n" +
			"\tGetFiddleValues() ( []float64 )\n" +
			"\tFixFiddleValue( int, float64 ) ( float64 )\n" +
			"\tSetFiddleValues( []float64 )\n\n" +
			"\tGetNumOutputValues() ( int )\n" +
			"\tGo( int ) ( []float64 )\n" +
			"\tGoSingle( int ) ( float64 )\n" +
			"}\n",
	); err != nil {
		panic(err)
	}
}

func writeToStatsFile(data string) {
	if _, err := statsOutputFile.WriteString(data); err != nil {
		panic(err)
	}
}

func createStatsFile() {
	writeToStatsFile(
		"package " + kLibraryName + "\n\n",
	)

	maxFiddleValues := 0
	maxOutputValues := 0
	lenFunctionArray := 0
	lenTimePeriodArray := 0

	groups := getGroups()
	for _, group := range groups {
		functions := getFunctions(group)
		for _, function := range functions {

			found := false
			for _, bannedFunction := range bannedFunctions {
				if function == bannedFunction {
					found = true
				}
			}

			if !found {
				handle := getFunctionHandle(function)
				info := getFunctionInfo(handle)

				if !shouldBeInFunctionArray(info) && !shouldBeInTimePeriodArray(info) {
					continue
				}

				if int(info.nbOptInput) > maxFiddleValues {
					maxFiddleValues = int(info.nbOptInput)
				}

				if int(info.nbOutput) > maxOutputValues {
					maxOutputValues = int(info.nbOutput)
				}

				if shouldBeInFunctionArray(info) {
					lenFunctionArray++
				}

				if shouldBeInTimePeriodArray(info) {
					lenTimePeriodArray++
				}
			}
		}
	}

	writeToStatsFile(
		"const (\n" +
			fmt.Sprintf("\tMaxFiddleValues = %v\n", maxFiddleValues) +
			fmt.Sprintf("\tMaxOutputValues = %v\n", maxOutputValues) +
			fmt.Sprintf("\tNumFunctions = %v\n", lenFunctionArray) +
			fmt.Sprintf("\tNumTimePeriodFunctions = %v\n", lenTimePeriodArray) +
			")",
	)
}

const (
	kBindingsFilename        = "gotalib/bindings.go"
	kFunctionArrayFilename   = "gotalib/function_array.go"
	kTimePeriodArrayFilename = "gotalib/time_period_array.go"
	kTaFunctionFilename      = "gotalib/ta_function.go"
	kStatsOutputFilename     = "gotalib/ta_stats.go"

	kLibraryName = "gotalib"
)

var (
	bindingsOutputFile        *os.File
	functionArrayOutputFile   *os.File
	timePeriodArrayOutputFile *os.File
	taFunctionOutputFile      *os.File
	statsOutputFile           *os.File

	bannedFunctions = []string{
		"TRIX",
	}
)

func init() {
	var err error
	bindingsOutputFile, err = os.OpenFile(kBindingsFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}

	functionArrayOutputFile, err = os.OpenFile(kFunctionArrayFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}

	timePeriodArrayOutputFile, err = os.OpenFile(kTimePeriodArrayFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}

	taFunctionOutputFile, err = os.OpenFile(kTaFunctionFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}

	statsOutputFile, err = os.OpenFile(kStatsOutputFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
}

func main() {

	initBindings()
	initFunctionArray()
	initTimePeriodArray()

	// handle := getFunctionHandle( "CDLINVERTEDHAMMER" )
	// info := getFunctionInfo( handle )
	// createBinding( info )
	// createFunctionArray( info )
	// createTimePeriodArray( info )

	groups := getGroups()
	for _, group := range groups {
		functions := getFunctions(group)
		for _, function := range functions {

			found := false
			for _, bannedFunction := range bannedFunctions {
				if function == bannedFunction {
					found = true
				}
			}

			if !found {
				handle := getFunctionHandle(function)
				info := getFunctionInfo(handle)

				createBinding(info)
				createFunctionArray(info)
				createTimePeriodArray(info)
			}
		}
	}

	shutdownFunctionArray()
	shutdownTimePeriodArray()

	bindingsOutputFile.Close()
	functionArrayOutputFile.Close()
	timePeriodArrayOutputFile.Close()

	createTaFunctionFile()
	taFunctionOutputFile.Close()

	createStatsFile()
	statsOutputFile.Close()
}
