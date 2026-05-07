package models

// UserIn — входная модель с валидацией (Task 10.2, аналог Pydantic-модели)
// username: required string
// age:      required int, > 18  (аналог conint(gt=18))
// email:    required, валидный e-mail (аналог EmailStr)
// password: required string, длина 8–16 (аналог constr(min_length=8, max_length=16))
// phone:    необязательное поле (аналог Optional[str] = 'Unknown')
type UserIn struct {
	Username string `json:"username" binding:"required"`
	Age      int    `json:"age"      binding:"required,gt=18"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=16"`
	Phone    string `json:"phone"`
}

// UserOut — ответная модель (пароль не возвращается)
type UserOut struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Age      int    `json:"age"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}
