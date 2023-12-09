package types

type LitmusType string

const (
	LitmusPodDelete   LitmusType = "litmus-pod-delete"
	LitmusPodIoStress LitmusType = "litmus-pod-io-stress"
)
