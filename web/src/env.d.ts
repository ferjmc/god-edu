/// <reference types="astro/client" />

interface ImportMetaEnv {
	/** Origin of the Go API, e.g. http://localhost:8080 in dev. */
	readonly PUBLIC_API_URL?: string;
}

interface ImportMeta {
	readonly env: ImportMetaEnv;
}
