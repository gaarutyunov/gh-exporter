package binpack

type Packable interface {
	Size() uint64
}

func FirstFit(items []Packable, cap uint64) (bins [][]Packable, remainder []Packable) {
	binSlice := []uint64{cap}

	for i := 0; i < len(items); i++ {
		settled := false
		j := 0

		for !settled && j < len(binSlice) {
			if items[i].Size() <= binSlice[j] {
				binSlice[j] -= items[i].Size()
				if len(bins) < j+1 {
					bins = append(bins, []Packable{items[i]})
				} else {
					bins[j] = append(bins[j], items[i])
				}
				settled = true
			} else {
				j++
			}
		}

		if !settled && items[i].Size() <= cap {
			binSlice = append(binSlice, cap-items[i].Size())
			bins = append(bins, []Packable{items[i]})
		} else if !settled {
			remainder = append(remainder, items[i])
		}
	}

	return
}
