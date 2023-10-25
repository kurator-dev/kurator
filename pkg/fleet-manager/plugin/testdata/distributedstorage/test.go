/*
Copyright Kurator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

func ToMap(obj interface{}) map[string]interface{} {

	m := make(map[string]interface{})

	v := reflect.ValueOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {

		f := v.Field(i)
		ft := t.Field(i)
		//fmt.Println(f, ft)

		var fp reflect.Value

		if f.Kind() == reflect.Ptr && !f.IsNil() {
			fmt.Println(1)
			fp = f
			f = fp.Elem()
			m[ft.Name] = fp.Elem()
			fmt.Println(fp.Elem().Type(), f)
		}
		//fmt.Println(f, fp)
		//fmt.Println(ft.Name, fp.Elem())
		// if f.IsNil() {
		// 	m[ft.Name] = fp.Elem().Interface()
		// }
		fmt.Println(f.Kind())
		// m[ft.Name] = fp
		fmt.Println(m)
		if f.Kind() == reflect.Struct {
			fmt.Println(2)
			mp := ToMap(f.Interface())
			m[ft.Name] = mp
			fmt.Println(ft.Name, mp, m)
		}
	}
	fmt.Println(3, m, reflect.TypeOf(m["DataDirHostPath"]))
	return m
}
