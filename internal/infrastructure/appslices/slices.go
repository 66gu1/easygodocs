package appslices

func Map[T any, DTO any](items []T, mapper func(T) DTO) []DTO {
	resp := make([]DTO, len(items))
	for i, item := range items {
		resp[i] = mapper(item)
	}
	return resp
}
