package api

import (
	"fmt"
	"reflect"
)

type diffValues struct {
	Current  interface{}
	Proposed interface{}
}

type diff struct {
	ID          string
	Param       map[string]diffValues
	Composition map[string]diffComposition
}

func (d *diff) Empty() bool {
	return len(d.Composition) == 0 && len(d.Param) == 0
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

//hasIdentifierType used to check if other type implements the HasIdentifier interface
var hasIdentifierType = reflect.TypeOf((*HasIdentifier)(nil)).Elem()

//identifierFormInterface retrieve the HasIdentifier interface from a generic interface
func identifierFormInterface(i interface{}) (HasIdentifier, error) {
	if i == nil {
		return nil, fmt.Errorf("nil interface cannot get identifier")
	}
	iHasIdentifier, ok := i.(HasIdentifier)
	if !ok {
		return nil, fmt.Errorf("type assertion to 'hasIdentifier' failed")
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return nil, fmt.Errorf("nil pointed value")
		}
	}
	return iHasIdentifier, nil
}

func checkDiff2(current, proposed HasIdentifier) (*diff, error) {
	//Inputs validation
	if current == nil || proposed == nil {
		return nil, fmt.Errorf("Nil inputs")
	}
	if reflect.TypeOf(current).Name() != reflect.TypeOf(proposed).Name() {
		return nil, fmt.Errorf("diff on object of different type")
	}
	if current.ID() != proposed.ID() {
		return nil, fmt.Errorf("diff on object with different ID")
	}

	//Prepare output
	d := diff{ID: current.ID(), Param: map[string]diffValues{}, Composition: map[string]diffComposition{}}

	//Get the Value out of the inputs
	vc := reflect.ValueOf(current)
	vp := reflect.ValueOf(proposed)
	if vc.Type().Kind() == reflect.Interface {
		vc = vc.Elem()
		vp = vp.Elem()
	}
	//Test in sequence with interface in case the interface value is a Ptr
	if vc.Type().Kind() == reflect.Ptr {
		vc = vc.Elem()
		vp = vp.Elem()
	}

	for i := 0; i < vc.NumField(); i++ {
		valueFieldc := vc.Field(i)
		typeFieldc := vc.Type().Field(i)
		tagc := typeFieldc.Tag
		if tagc.Get("diff") == "ignore" {
			//The field was tagged to be ignored in the diff process
			continue
		}
		fieldName := typeFieldc.Name
		valueFieldp := vp.FieldByName(fieldName)

		k := valueFieldc.Type().Kind()
		switch {
		case k >= reflect.Bool && k <= reflect.Complex128, k == reflect.String:
			if !reflect.DeepEqual(valueFieldc.Interface(), valueFieldp.Interface()) {
				d.Param[fieldName] = diffValues{Current: valueFieldc, Proposed: valueFieldp}
			}
		case k == reflect.Interface, k == reflect.Struct, k == reflect.Ptr:
			if valueFieldc.Type().Implements(hasIdentifierType) {

				cID, cErr := identifierFormInterface(valueFieldc.Interface())
				pID, pErr := identifierFormInterface(valueFieldp.Interface())

				if cErr != nil && pErr != nil {
					// both nil, not initialized
					break
				}

				//nil current and new proposed
				if cErr != nil && pErr == nil {
					dc := d.Composition[fieldName]
					dc.New = append(dc.New, pID)
					d.Composition[fieldName] = dc
					break
				}

				//valid current and nil proposed
				if cErr == nil && pErr != nil {
					dc := d.Composition[fieldName]
					dc.Deleted = append(dc.Deleted, cID)
					d.Composition[fieldName] = dc
					break
				}

				//two valid values. Need to compare
				if cErr == nil && pErr == nil {
					//same identifier need to compare content
					if cID.ID() == pID.ID() {
						md, err := checkDiff2(cID, pID)
						if err != nil {
							return nil, err
						}
						if !md.Empty() {
							dc := d.Composition[fieldName]
							dc.Modified = append(dc.Modified, *md)
							d.Composition[fieldName] = dc
						}
					} else { //the object was replaced by another one
						dc := d.Composition[fieldName]
						dc.Deleted = append(dc.Deleted, cID)
						dc.New = append(dc.New, pID)
						d.Composition[fieldName] = dc
					}
				}
			} else {
				if !reflect.DeepEqual(valueFieldc.Interface(), valueFieldp.Interface()) {
					d.Param[fieldName] = diffValues{Current: valueFieldc, Proposed: valueFieldp}
				}
			}
		case k == reflect.Array || k == reflect.Slice:
			// check if inner type implements HasIdentifier
			if valueFieldc.Type().Elem().Implements(hasIdentifierType) {
				same, added, deleted, err := checkDiffInComposition(valueFieldc.Interface(), valueFieldp.Interface())
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
					md, err := checkDiff2(n[0], n[1])
					if err != nil {
						return nil, err
					}
					if !md.Empty() {
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
		}
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
