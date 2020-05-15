package oconfig

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
)

func UnMarshal(data []byte,result interface{}) (err error) {
	t := reflect.TypeOf(result)
	v := reflect.ValueOf(result)
	_ = v
	kind := t.Kind()

	//输入校验
	if kind != reflect.Ptr {
		panic("please input a address")
	}
	//解析配置文件
	var sectionName string
	lines := strings.Split(string(data),"\n")
	lineNo := 0
	for _,line := range lines {
		lineNo ++
		//去掉首位空串
		line = strings.Trim(line,"\t\n\r")
		if len(line) == 0 {
			//空行处理
			continue
		}
		//去掉注释
		if line[0] == '#' || line[0] == ';' {
			continue
		}
		// 解析group
		if line[0] == '[' {
			//解析group的名称,是否以[]包含校验
			if len(line)<=2 || line[len(line)-1] != ']' {
				tips := fmt.Sprintf("syntax err,invalid section:\"%s\",line:%d",line,lineNo)
				panic(tips)
			}
			//获取group名称,去空格
			sectionName = strings.TrimSpace(line[1:len(line)-1])
			if len(sectionName) == 0 {
				tips := fmt.Sprintf("syntax err,invalid section:\"%s\",line:%d",line,lineNo)
				panic(tips)
			}
		}else {
			//group的字段
			if len(sectionName) == 0 {
				tips := fmt.Sprintf("key-value:%s 不属于任何group,lineNo:%d",line,lineNo)
				panic(tips)
			}
			//找到第一个等于号所在的位置
			index := strings.Index(line,"=")
			if index == -1 {
				//等号之前没有配置的异常
				tips := fmt.Sprintf("syntax error,not found =,line:%s,lineNo:%d",line,lineNo)
				panic(tips)
			}
			key := strings.TrimSpace(line[0:index])  //去掉空格
			value := strings.TrimSpace(line[index+1:])  //获取到的value
			if len(key) == 0 {
				//key不存在异常
				tips := fmt.Sprintf("syntax error,not found key,line:%s,lineNo:%d",line,lineNo)
				panic(tips)
			}
			//通过Config找到所属的SectionName配置，需要查找两遍
			//1.找到sectionName在result中对应的结构体s1
			for i := 0;i < t.Elem().NumField();i++ {
				tfield := t.Elem().Field(i)
				vField := v.Elem().Field(i)
				if tfield.Tag.Get("ini") != sectionName {
					continue
				}
				//2.通过当前解析的key，找到对应的结构体s1中的对应字段
				tfieldType := tfield.Type
				if tfieldType.Kind() != reflect.Struct {
					tips := fmt.Sprintf("field %s is not a struct",tfield.Name)
					panic(tips)
				}
				//查找子结构体中的数据
				for j:=0;j<tfieldType.NumField();j++ {
					//找不到key对应field的时候就跳过
					tKeyField := tfieldType.Field(j)
					vKeyField := vField.Field(j)
					if tKeyField.Tag.Get("ini") != key {
						continue
					}
					//找到了对应的字段,并将值映射到对应的字段
					switch tKeyField.Type.Kind() {
					case reflect.String:
						vKeyField.SetString(value)
					case reflect.Int,reflect.Uint,reflect.Int16,reflect.Uint16:
						fallthrough   //穿透到下个case
					case reflect.Int32,reflect.Uint32,reflect.Int64,reflect.Uint64:
						valueInt,err := strconv.ParseInt(value,10,64) //十进制64字节int
						if err != nil {
							tips := fmt.Sprintf("value %s is not a convert to int,lineNo:%d",value,lineNo)
							panic(tips)
						}
						vKeyField.SetInt(valueInt)
					case reflect.Float32,reflect.Float64:
						valueFloat,err := strconv.ParseFloat(value,64)
						if err != nil {
							tips := fmt.Sprintf("value %s is not a convert to float,lineNo:%d",value,lineNo)
							panic(tips)
						}
						vKeyField.SetFloat(valueFloat)
					default:
						tips := fmt.Sprintf("key \"%s\" is not a convert to %v,lineNo:%d",key,tKeyField.Type.Kind(),lineNo)
						panic(tips)
					}
				}
				break
			}
		}
	}
	return
}

func UnMarshalFile(filename string,result interface{}) (err error) {
	//读取配置文件
	data,err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	return UnMarshal(data,result)
}

func Marshal(result interface{}) (data []byte,err error) {
	//序列化用户数据
	t := reflect.TypeOf(result)
	v := reflect.ValueOf(result)

	if t.Kind() != reflect.Struct {
		panic("please input struct type")
	}

	var strSlice []string
	//遍历结构体中的字段
	for i:=0;i<t.NumField();i++ {
		tField := t.Field(i)
		vField := v.Field(i)
		if tField.Type.Kind() != reflect.Struct {
			continue  //非结构体跳过
		}

		//sectionName的获取
		sectionName := tField.Name
		//subTFieldTagName := subTField.Tag.Get("ini")
		if len(tField.Tag.Get("ini")) > 0 {
			sectionName = tField.Tag.Get("ini")
		}
		//使用[]拼接sectionName
		sectionName = fmt.Sprintf("[%s]\n",sectionName)
		//保存sectionName信息
		strSlice = append(strSlice,sectionName)
		for j:=0;j<tField.Type.NumField();j++ {
			//1. 拿到类型信息
			subTField := tField.Type.Field(j)   //获取字段名称
			if subTField.Type.Kind() == reflect.Struct || subTField.Type.Kind() == reflect.Ptr{
				continue   //跳过结构体或者指针类型，正常是key=value格式
			}
			subTFieldName := subTField.Name
			subTFieldTagName := subTField.Tag.Get("ini")
			if len(subTFieldTagName) > 0 {
				subTFieldName = subTFieldTagName
			}
			//2. 拿到值信息
			subVField := vField.Field(j)
			fieldStr := fmt.Sprintf("%s=%v\n",subTFieldName,subVField.Interface())
			//fmt.Printf("conf%s\n",fieldStr)
			strSlice = append(strSlice,fieldStr) //保存sectionName对应的字段
		}
	}
	for _, v := range strSlice {
		//展开slice的结果
		data = append(data,[]byte(v)...)
	}
	return
}

func MarshalFile(filename string,result interface{}) (err error) {
	//将用户输入的数据写入配置文件
	data,err := Marshal(result)
	if err  != nil {
		return
	}
	err = ioutil.WriteFile(filename,data,0755)
	return
}