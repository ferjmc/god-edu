import { apiFetch } from "./client";

/** Mirrors models.UserRole in the Go API. */
export type UserRole = "ADMIN" | "COMMUNITY_MEMBER" | "PUBLIC_MEMBER" | "PAID_MEMBER";

/** Mirrors handlers.userResponse in the Go API — never includes the password hash. */
export type User = {
	id: number;
	email: string;
	name: string;
	auth_provider: "email" | "google" | "facebook";
	email_verified: boolean;
	role: UserRole;
	needs_birth_date: boolean;
};

export function register(input: {
	email: string;
	password: string;
	name: string;
	accepted_terms: boolean;
	birth_date: string;
}): Promise<User> {
	return apiFetch<User>("/auth/register", { method: "POST", body: input });
}

/** Guarda la fecha de nacimiento del usuario autenticado — la usa
 * /completar-perfil, a la que el callback de OAuth redirige cuando falta
 * este dato (alta social nueva o cuenta OAuth vieja sin completar). */
export function updateBirthDate(birthDate: string): Promise<void> {
	return apiFetch<void>("/auth/birth-date", { method: "PATCH", body: { birth_date: birthDate } });
}

export function login(input: { email: string; password: string }): Promise<User> {
	return apiFetch<User>("/auth/login", { method: "POST", body: input });
}

export function logout(): Promise<void> {
	return apiFetch<void>("/auth/logout", { method: "POST" });
}

/** Resolves the current session's user, or null if there isn't one (401). */
export async function me(): Promise<User | null> {
	try {
		return await apiFetch<User>("/auth/me");
	} catch (err) {
		if (err instanceof Error && err.name === "ApiError" && (err as { status?: number }).status === 401) {
			return null;
		}
		throw err;
	}
}

/** Consume el token del link de verificación (?token= en la URL del email).
 * POST a propósito en el API, no GET — ver comentario en
 * AuthHandler.VerifyEmail sobre escáneres de seguridad que "clickean" links
 * de emails en automático. */
export function verifyEmail(token: string): Promise<{ email_verified: boolean }> {
	return apiFetch<{ email_verified: boolean }>("/auth/verify-email", { method: "POST", body: { token } });
}

/** Pide que se reenvíe el link de verificación. La respuesta es siempre el
 * mismo mensaje genérico exista o no la cuenta — no revela nada por diseño,
 * ver AuthHandler.ResendVerification. */
export function resendVerification(email: string): Promise<{ message: string }> {
	return apiFetch<{ message: string }>("/auth/resend-verification", { method: "POST", body: { email } });
}

/** Pide el link de restablecimiento de contraseña. Mismo patrón que
 * resendVerification: respuesta genérica exista o no el email, ver
 * AuthHandler.ForgotPassword. */
export function forgotPassword(email: string): Promise<{ message: string }> {
	return apiFetch<{ message: string }>("/auth/forgot-password", { method: "POST", body: { email } });
}

/** Consume el token del link de restablecimiento (?token= en la URL del
 * email, ver AuthHandler.sendPasswordResetEmail) y fija la contraseña nueva. */
export function resetPassword(token: string, newPassword: string): Promise<{ status: string }> {
	return apiFetch<{ status: string }>("/auth/reset-password", {
		method: "POST",
		body: { token, new_password: newPassword },
	});
}
