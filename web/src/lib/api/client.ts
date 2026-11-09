/**
 * Base REST client. Every call to the API goes through this file — no
 * `.astro`/`.tsx` component ever calls `fetch` directly.
 *
 * Resolves the API origin once, always sends the session cookie
 * (`credentials: "include"`, required since the cookie is httpOnly and the
 * API lives on a different origin/port than the site), and normalizes
 * errors into a single ApiError type.
 */

const API_BASE_URL = import.meta.env.PUBLIC_API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
	status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = "ApiError";
		this.status = status;
	}
}

type RequestOptions = {
	method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
	body?: unknown;
};

const REQUEST_TIMEOUT_MS = 15_000;
const UPLOAD_TIMEOUT_MS = 60_000;

/** Común a apiFetch y apiUpload: 204 sin cuerpo, error normalizado a
 * ApiError, o el JSON de la respuesta. El mensaje nunca queda vacío: sobre
 * HTTP/2 `statusText` siempre es "", y los componentes tratan un mensaje
 * vacío como "sin error". */
async function handleResponse<T>(response: Response): Promise<T> {
	if (response.status === 204) {
		return undefined as T;
	}

	const data = await response.json().catch(() => null);

	if (!response.ok) {
		const message = (data && typeof data.error === "string" && data.error) || response.statusText || `Error ${response.status}`;
		throw new ApiError(response.status, message);
	}

	if (data === null) {
		throw new ApiError(response.status, "Respuesta inválida del servidor.");
	}

	return data as T;
}

/** Normaliza fallas de red y timeouts a ApiError (status 0), para que los
 * componentes no tengan que distinguir un TypeError de fetch. */
async function send(url: string, init: RequestInit, timeoutMs: number): Promise<Response> {
	try {
		return await fetch(url, { ...init, signal: AbortSignal.timeout(timeoutMs) });
	} catch (err) {
		if (err instanceof DOMException && err.name === "TimeoutError") {
			throw new ApiError(0, "El servidor tardó demasiado en responder.");
		}
		throw new ApiError(0, "No pudimos conectar con el servidor.");
	}
}

/** Low-level request helper. Prefer the typed functions in the domain files
 * (auth.ts, courses.ts, ...) over calling this directly. */
export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const response = await send(
		`${API_BASE_URL}${path}`,
		{
			method: options.method ?? "GET",
			credentials: "include",
			headers: options.body ? { "Content-Type": "application/json" } : undefined,
			body: options.body ? JSON.stringify(options.body) : undefined,
		},
		REQUEST_TIMEOUT_MS,
	);

	return handleResponse<T>(response);
}

/** Subida multipart — la usa solo POST .../content/pdf, el único endpoint
 * que recibe un archivo en vez de JSON. No seteamos Content-Type a mano: el
 * browser arma el boundary del multipart solo si se lo dejamos hacerlo. */
export async function apiUpload<T>(path: string, formData: FormData): Promise<T> {
	const response = await send(
		`${API_BASE_URL}${path}`,
		{
			method: "POST",
			credentials: "include",
			body: formData,
		},
		UPLOAD_TIMEOUT_MS,
	);

	return handleResponse<T>(response);
}
