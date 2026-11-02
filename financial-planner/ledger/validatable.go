package main

// Validatable задаёт контракт проверки валидности данных структуры.
type Validatable interface {
	Validate() error
}

// CheckValid проверяет любой объект, реализующий Validatable.
func CheckValid(v Validatable) error {
	return v.Validate()
}
