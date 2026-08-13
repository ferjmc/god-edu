// Package storage habla con Cloudflare R2 (compatible S3) para los PDFs
// de las lecciones. Usa el SDK oficial de AWS en vez de armar requests
// firmados a mano — escribir la firma SigV4 por nuestra cuenta es mucho
// más de 30 minutos de trabajo y un lugar fácil para introducir un bug de
// seguridad.
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Config son las credenciales del bucket de R2. AccountID arma el
// endpoint (R2 no tiene un endpoint fijo como S3, depende de la cuenta).
// PublicBaseURL es el dominio público desde el que se sirven los objetos
// (el dominio custom o el *.r2.dev que Cloudflare da al hacer público el
// bucket) — separado del endpoint de la API, que es privado.
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBaseURL   string
}

// R2 es un cliente delgado sobre el bucket configurado. Un solo bucket
// para toda la plataforma alcanza para este tamaño — no hay necesidad de
// manejar múltiples buckets ni credenciales por curso.
type R2 struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
}

func NewR2(cfg R2Config) (*R2, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("auto"), // R2 no tiene regiones; "auto" es lo que Cloudflare documenta
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: cargando config de AWS SDK: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &R2{client: client, bucket: cfg.Bucket, publicBaseURL: cfg.PublicBaseURL}, nil
}

// Upload sube un objeto y devuelve su URL pública. key es la ruta dentro
// del bucket (ej. "lessons/42/guia.pdf") — la arma el caller, no acá, para
// que quede claro desde el handler cómo se organiza el bucket.
func (r *R2) Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("storage: subiendo %q: %w", key, err)
	}

	return fmt.Sprintf("%s/%s", r.publicBaseURL, key), nil
}
