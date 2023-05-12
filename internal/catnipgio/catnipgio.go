package catnipgio

const ScalingWindow = 1.5 // seconds
const PeakThreshold = 0.01
const ZeroThreshold = 5

type DrawStyle int

const (
	DrawVertically DrawStyle = iota
	DrawLines
)

func calculateBar(value, height float64) float64 {
	bar := min(value, height)
	return height - bar
}

func max[T ~int | ~float64](i, j T) T {
	if i > j {
		return i
	}
	return j
}

func min[T ~int | ~float64](i, j T) T {
	if i < j {
		return i
	}
	return j
}
