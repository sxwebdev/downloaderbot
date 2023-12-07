package s3

type Config struct {
	AccessID   string `validate:"required" secret:"true"`
	SecretKey  string `validate:"required" secret:"true"`
	BucketName string `validate:"required"`
	Token      string `secret:"true" usage:"used only for amazon s3"`
	Region     string `validate:"required" example:"ru-1"`
	Endpoint   string `validate:"required"`
}
