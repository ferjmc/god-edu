import { apiFetch } from "./client";

/** Mirrors handlers.certificateResponse in the Go API. */
export type Certificate = {
	recipientName: string;
	courseTitle: string;
	issuedAt: string;
};

/** Verificación pública de un certificado por su código (el que viaja en la
 * URL y en el QR). El endpoint no requiere sesión — cualquiera con el link
 * puede confirmar si un certificado es válido y a quién fue emitido. */
export function verifyCertificate(code: string): Promise<Certificate> {
	return apiFetch<Certificate>(`/certificates/${encodeURIComponent(code)}`);
}
