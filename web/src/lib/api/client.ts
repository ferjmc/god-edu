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
	method?: "GET" | "POST" | "PUT" | "DELETE";
	body?: unknown;
};

/** Low-level request helper. Prefer the typed functions in the domain files
 * (auth.ts, courses.ts, ...) over calling this directly. */
export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const response = await fetch(`${API_BASE_URL}${path}`, {
		method: options.method ?? "GET",
		credentials: "include",
		headers: options.body ? { "Content-Type": "application/json" } : undefined,
		body: options.body ? JSON.stringify(options.body) : undefined,
	});

	if (response.status === 204) {
		return undefined as T;
	}

	const data = await response.json().catch(() => null);

	if (!response.ok) {
		const message = data && typeof data.error === "string" ? data.error : response.statusText;
		throw new ApiError(response.status, message);
	}

	return data as T;
}
