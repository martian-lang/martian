package syntax

func (valExp *ValExp) ToInterface() interface{} {
	// Convert tree of Exps into a tree of interface{}s.
	if valExp.Kind == "array" {
		varray := []interface{}{}
		for _, exp := range valExp.Value.([]Exp) {
			varray = append(varray, exp.ToInterface())
		}
		return varray
	} else if valExp.Kind == "map" {
		vmap := map[string]interface{}{}
		// Type assertion fails if map is empty
		valExpMap, ok := valExp.Value.(map[string]Exp)
		if ok {
			for k, exp := range valExpMap {
				vmap[k] = exp.ToInterface()
			}
		}
		return vmap
	} else {
		return valExp.Value
	}
}

func (self *RefExp) ToInterface() interface{} {
	return nil
}
