package api

import (
	"fmt"
	"reflect"
)

type diffValues struct {
	Current  interface{}
	Proposed interface{}
}

type diffComposition struct {
	Modified []diff
	Deleted  []interface{}
	New      []interface{}
}

//HasIdentifier object implementing this interface are uniquely indentified by their path
type HasIdentifier interface {
	ID() string
}

var hasIdentifierType = reflect.TypeOf((*HasIdentifier)(nil)).Elem()

func (d *diff) Empty() bool {
	return len(d.Composition) == 0 && len(d.Param) == 0
}

type diff struct {
	ID          string
	Param       map[string]diffValues
	Composition map[string]diffComposition
}

func newDiffComposition() diffComposition {
	return diffComposition{[]diff{}, []interface{}{}, []interface{}{}}
}

func checkDiff2(current, proposed HasIdentifier) (*diff, error) {
	if current == nil || proposed == nil {
		return nil, fmt.Errorf("Nil inputs")
	}

	if reflect.TypeOf(current).Name() != reflect.TypeOf(proposed).Name() {
		return nil, fmt.Errorf("diff on object of different type")
	}

	if current.ID() != proposed.ID() {
		return nil, fmt.Errorf("diff on object with different ID")
	}

	d := diff{ID: current.ID(), Param: map[string]diffValues{}, Composition: map[string]diffComposition{}}

	if &current == &proposed {
		return &d, nil
	}

	vc := reflect.ValueOf(current)
	vp := reflect.ValueOf(proposed)
	for i := 0; i < vc.NumField(); i++ {

		valueFieldc := vc.Field(i)
		typeFieldc := vc.Type().Field(i)
		tagc := typeFieldc.Tag
		if tagc.Get("diff") == "ignore" {
			continue
		}
		fieldName := typeFieldc.Name

		valueFieldp := vp.FieldByName(fieldName)

		fmt.Printf("%s of kind %s\n", fieldName, valueFieldc.Type().Kind().String())
		k := valueFieldc.Type().Kind()
		switch {
		case k >= reflect.Bool && k <= reflect.Complex128 || k == reflect.String:
			if !reflect.DeepEqual(valueFieldc.Interface(), valueFieldp.Interface()) {
				d.Param[fieldName] = diffValues{Current: valueFieldc, Proposed: valueFieldp}
			}
		case k == reflect.Array || k == reflect.Slice:

			// check if inner type implements HasIdentifier
			if valueFieldc.Type().Elem().Implements(hasIdentifierType) {
				same, added, deleted, err := checkDiffInComposition(valueFieldc.Interface(), valueFieldp.Interface())
				d.Composition[fieldName] = newDiffComposition()
				if err != nil {
					return nil, err
				}
				for _, n := range added {
					dc := d.Composition[fieldName]
					dc.New = append(dc.New, n)
					d.Composition[fieldName] = dc
				}
				for _, n := range deleted {
					dc := d.Composition[fieldName]
					dc.Deleted = append(dc.Deleted, n)
					d.Composition[fieldName] = dc
				}
				for _, n := range same {
					fmt.Printf("Checking compo under same path %s\n", n[0].ID())
					md, err := checkDiff(n[0], n[1])
					if err != nil {
						return nil, err
					}
					if !md.Empty() {
						fmt.Println("Diff detected in composition")
						dc := d.Composition[fieldName]
						dc.Modified = append(dc.Modified, *md)
						d.Composition[fieldName] = dc
					}
				}
			} else {
				if !reflect.DeepEqual(valueFieldc.Interface(), valueFieldp.Interface()) {
					d.Param[fieldName] = diffValues{Current: valueFieldc, Proposed: valueFieldp}
				}
			}

			fmt.Printf("%s of elment in slice %s\n", fieldName, valueFieldc.Type().Elem().Kind().String())

			fmt.Printf("%s implement hasIdentifier:%v\n", fieldName, valueFieldc.Type().Elem().Implements(hasIdentifierType))

		}
		// Bool
		// Int
		// Int8
		// Int16
		// Int32
		// Int64
		// Uint
		// Uint8
		// Uint16
		// Uint32
		// Uint64
		// Uintptr
		// Float32
		// Float64
		// Complex64
		// Complex128
		// Array
		// Chan
		// Func
		// Interface
		// Map
		// Ptr
		// Slice
		// String
		// Struct
		// UnsafePointer

		// av := vc.Field(i)
		// bv := vp.FieldByName(av.Name)

		// at := av.Type()
		// bt := bv.Type()
		// if at != bt {
		// 	w.printf("%v != %v", at, bt)
		// 	return
		// }

		// // numeric types, including bool
		// if at.Kind() < reflect.Array {
		// 	a, b := av.Interface(), bv.Interface()
		// 	if a != b {
		// 		w.printf("%#v != %#v", a, b)
		// 	}
		// 	return
		// }

	}
	return &d, nil
}

func checkDiff(current, proposed HasIdentifier) (*diff, error) {
	if current == nil || proposed == nil {
		return nil, fmt.Errorf("Nil inputs")
	}

	if current.ID() != proposed.ID() {
		return nil, fmt.Errorf("diff on bucket not under same path")
	}

	d := diff{ID: current.ID(), Param: map[string]diffValues{}, Composition: map[string]diffComposition{}}

	vc := reflect.ValueOf(current)
	vp := reflect.ValueOf(proposed)
	for i := 0; i < vc.NumField(); i++ {

		vcurrent := vc.Field(i).Interface()
		vproposed := vp.Field(i).Interface()

		tag := vc.Type().Field(i).Tag
		fieldName := vc.Type().Field(i).Name

		switch tag.Get("diff") {
		case "value":
			if !reflect.DeepEqual(vcurrent, vproposed) {
				d.Param[fieldName] = diffValues{Current: vcurrent, Proposed: vproposed}
			}
		case "composition":
			same, added, deleted, err := checkDiffInComposition(vcurrent, vproposed)
			d.Composition[fieldName] = newDiffComposition()
			if err != nil {
				return nil, err
			}
			for _, n := range added {
				dc := d.Composition[fieldName]
				dc.New = append(dc.New, n)
				d.Composition[fieldName] = dc
			}
			for _, n := range deleted {
				dc := d.Composition[fieldName]
				dc.Deleted = append(dc.Deleted, n)
				d.Composition[fieldName] = dc
			}
			for _, n := range same {
				fmt.Printf("Checking compo under same path %s\n", n[0].ID())
				md, err := checkDiff(n[0], n[1])
				if err != nil {
					return nil, err
				}
				if !md.Empty() {
					fmt.Println("Diff detected in composition")
					dc := d.Composition[fieldName]
					dc.Modified = append(dc.Modified, *md)
					d.Composition[fieldName] = dc
				}
			}
		}

	}
	return &d, nil
}

func checkDiffInComposition(current, proposed interface{}) (samePath [][2]HasIdentifier, newPath, deletedPath []HasIdentifier, err error) {
	samePath = [][2]HasIdentifier{}
	newPath = []HasIdentifier{}
	deletedPath = []HasIdentifier{}

	// index all path in current composition
	currentMap := map[string]HasIdentifier{}
	s := reflect.ValueOf(current)
	for i := 0; i < s.Len(); i++ {
		item := s.Index(i)
		p, ok := item.Interface().(HasIdentifier)
		if !ok {
			err = fmt.Errorf("Compisition of non-PathIdentifier in current: %T", item.Interface())
			return
		}
		currentMap[p.ID()] = p
	}

	// index all path in proposed composition
	proposedMap := map[string]HasIdentifier{}
	s = reflect.ValueOf(proposed)
	for i := 0; i < s.Len(); i++ {
		item := s.Index(i)
		p, ok := item.Interface().(HasIdentifier)
		if !ok {
			err = fmt.Errorf("Compisition of non-PathIdentifier in proposed: %T", item.Interface())
			return
		}
		proposedMap[p.ID()] = p
	}

	//Deleted and Same
	for k := range currentMap {
		if p, ok := proposedMap[k]; ok {
			samePath = append(samePath, ([2]HasIdentifier{currentMap[k], p}))
		} else {
			deletedPath = append(deletedPath, currentMap[k])
		}
	}

	//New
	for k := range proposedMap {
		if _, ok := currentMap[k]; !ok {
			newPath = append(newPath, proposedMap[k])
		}
	}

	return samePath, newPath, deletedPath, nil
}
