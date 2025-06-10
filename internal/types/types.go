package types

type ScalerParams struct {
	MinReplicas        int32
	MaxReplicas        int32
	ScaleUpThreshold   int
	ScaleDownThreshold int
}
