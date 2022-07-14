package helper

import "fmt"

func SliceInterfacesToTypeSlice[K any](s []interface{}) ([]K, error) {
	var d []K
	for _, v := range s {
		if o, ok := v.(K); ok {
			d = append(d, o)
		} else {
			return nil, fmt.Errorf("%v is not an slice with same type", v)
		}
	}
	return d, nil
}
