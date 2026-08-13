import { atom } from "nanostores";
import * as authApi from "../lib/api/auth";
import type { User } from "../lib/api/auth";

export type AuthState =
	| { status: "loading" }
	| { status: "anonymous" }
	| { status: "authenticated"; user: User };

export const $auth = atom<AuthState>({ status: "loading" });

// Some pages mount more than one island that needs the session (e.g. the
// navbar's AuthStatus plus a page-level island like CourseDetail). Without
// this, each one would fire its own GET /auth/me on mount. inFlight makes
// them share the same request.
let inFlight: Promise<void> | null = null;

/** Resolves the current session against the API. Safe to call from every
 * island that needs auth state on mount — concurrent calls share one
 * request, and it's a no-op once $auth is past "loading". */
export function initAuth(): Promise<void> {
	if ($auth.get().status !== "loading") return Promise.resolve();
	if (!inFlight) {
		inFlight = authApi
			.me()
			.then((user) => {
				$auth.set(user ? { status: "authenticated", user } : { status: "anonymous" });
			})
			.finally(() => {
				inFlight = null;
			});
	}
	return inFlight;
}

export async function login(input: { email: string; password: string }): Promise<User> {
	const user = await authApi.login(input);
	$auth.set({ status: "authenticated", user });
	return user;
}

export async function register(input: { email: string; password: string; name: string }): Promise<User> {
	const user = await authApi.register(input);
	$auth.set({ status: "authenticated", user });
	return user;
}

export async function logout(): Promise<void> {
	await authApi.logout();
	$auth.set({ status: "anonymous" });
}

/** Vuelve a pedir /auth/me y actualiza el store, sin importar el estado
 * actual (a diferencia de initAuth, que es un no-op fuera de "loading").
 * Usado después de verificar el email: el usuario ya tiene sesión abierta,
 * pero email_verified cambió en el servidor y el store quedó desactualizado. */
export async function refreshAuth(): Promise<void> {
	const user = await authApi.me();
	$auth.set(user ? { status: "authenticated", user } : { status: "anonymous" });
}
