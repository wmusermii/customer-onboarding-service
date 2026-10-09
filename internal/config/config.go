package config

import "os"

type Config struct {
	HTTPPort     string
	DatabaseURL  string
	KafkaBrokers string
	NIKPepper    string

	// Dummy credentials untuk service eksternal
	EKYCBaseURL     string
	EKYCAPIKey      string
	AMLBaseURL      string
	AMLAPIKey       string
	CBSBaseURL      string
	CBSAPIKey       string
	EKYCDummyResult string // VERIFIED | REJECTED | MANUAL
}

func Load() Config {
	return Config{
		HTTPPort:        getenv("HTTP_PORT", "9200"),
		DatabaseURL:     getenv("DATABASE_URL", "postgres://komibank:komibank@localhost:5432/komibank?search_path=bankservices,public&sslmode=disable"),
		KafkaBrokers:    getenv("KAFKA_BROKERS", "localhost:9092"),
		NIKPepper:       getenv("NIK_PEPPER", "dummy-pepper-change-me"),
		EKYCBaseURL:     getenv("EKYC_BASE_URL", "http://localhost:9001"),
		EKYCAPIKey:      getenv("EKYC_API_KEY", "dummy-ekyc-key"),
		AMLBaseURL:      getenv("AML_BASE_URL", "http://localhost:9002"),
		AMLAPIKey:       getenv("AML_API_KEY", "dummy-aml-key"),
		CBSBaseURL:      getenv("CBS_BASE_URL", "http://localhost:9003"),
		CBSAPIKey:       getenv("CBS_API_KEY", "dummy-cbs-key"),
		EKYCDummyResult: getenv("EKYC_DUMMY_RESULT", "VERIFIED"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
