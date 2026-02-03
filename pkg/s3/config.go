package s3

type Config struct {
	AccessID   string `validate:"required" secret:"true" env:"S3_ACCESS_ID"`
	SecretKey  string `validate:"required" secret:"true" env:"S3_SECRET_KEY"`
	BucketName string `validate:"required" env:"S3_BUCKET_NAME"`
	Token      string `secret:"true" usage:"used only for amazon s3" env:"S3_TOKEN"`
	Region     string `validate:"required" example:"ru-1" env:"S3_REGION"`
	Endpoint   string `validate:"required" env:"S3_ENDPOINT"`
}
