import { useEffect, useState } from "react";
import { verifyEmail, resendVerification } from "../lib/api/auth";
import { refreshAuth } from "../stores/auth";
import { CheckCircleIcon, XCircleIcon } from "./icons";

type Status = "loading" | "missing" | "ready" | "verifying" | "success" | "error";

/** Lee ?token= de la URL (window.location, no props — esta página no tiene
 * ruta dinámica en Astro, el token viaja como query string) y consume
 * POST /auth/verify-email.
 *
 * La verificación se dispara con un click explícito, NO al cargar la página:
 * los escáneres de links de los proveedores de email (ej. Gmail) abren el
 * link en un navegador headless y ejecutan JS, así que un POST automático
 * gastaba el token de un solo uso antes de que el usuario llegara, y al
 * hacer clic veía "link vencido". Los escáneres no hacen click en botones. */
export default function VerifyEmailStatus() {
	const [status, setStatus] = useState<Status>("loading");
	const [token, setToken] = useState("");
	const [resendState, setResendState] = useState<"idle" | "sending" | "sent">("idle");
	const [resendEmail, setResendEmail] = useState("");

	useEffect(() => {
		const t = new URLSearchParams(window.location.search).get("token");
		if (!t) {
			setStatus("missing");
			return;
		}
		setToken(t);
		setStatus("ready");
	}, []);

	function handleVerify() {
		setStatus("verifying");
		verifyEmail(token)
			.then(() => {
				setStatus("success");
				// El usuario ya tiene sesión abierta desde el registro — refrescamos
				// el store para que refleje email_verified: true sin pedirle que
				// vuelva a loguearse.
				void refreshAuth();
			})
			.catch(() => setStatus("error"));
	}

	async function handleResend(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setResendState("sending");
		try {
			await resendVerification(resendEmail);
		} catch {
			// El endpoint no distingue errores reales de "no existe" — ver
			// AuthHandler.ResendVerification. Si igual falla la request en sí
			// (red caída, etc.), no tenemos mucho más que ofrecer acá.
		} finally {
			setResendState("sent");
		}
	}

	if (status === "loading") {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando" />;
	}

	if (status === "ready" || status === "verifying") {
		return (
			<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
				<p className="font-serif text-ink-80">Hacé clic en el botón para confirmar tu dirección de email.</p>
				<button type="button" onClick={handleVerify} disabled={status === "verifying"} className="btn btn-cta">
					{status === "verifying" ? "Confirmando…" : "Confirmar mi email"}
				</button>
			</div>
		);
	}

	if (status === "success") {
		return (
			<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
				<CheckCircleIcon className="h-10 w-10 text-liturgical-gold" />
				<div>
					<h2 className="font-display text-xl text-marian-blue-deep">¡Cuenta confirmada!</h2>
					<p className="mt-1 font-serif text-ink-80">Ya podés acceder a todo el contenido de tus cursos.</p>
				</div>
				<a href="/cursos" className="btn btn-cta">
					Ir a mis cursos
				</a>
			</div>
		);
	}

	if (status === "missing") {
		return (
			<div className="flex flex-col items-start gap-3 rounded-[4px] border border-ink-15 bg-paper p-8">
				<XCircleIcon className="h-10 w-10 text-sacred-red" />
				<p className="font-serif text-ink-80">
					Este link no trae ningún token de verificación. Revisá que hayas copiado la URL completa del email.
				</p>
			</div>
		);
	}

	// status === "error": token inválido, ya usado, o vencido (24hs).
	return (
		<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
			<XCircleIcon className="h-10 w-10 text-sacred-red" />
			<p className="font-serif text-ink-80">El link de verificación es inválido o venció. Pedí que te manden uno nuevo:</p>

			{resendState === "sent" ? (
				<p className="font-ui text-sm text-ink-80">Si el email existe y no fue verificado, te mandamos un nuevo link.</p>
			) : (
				<form onSubmit={handleResend} className="flex w-full max-w-sm flex-col gap-3">
					<input
						type="email"
						required
						placeholder="tu@email.com"
						value={resendEmail}
						onChange={(e) => setResendEmail(e.target.value)}
						className="input input-bordered w-full"
					/>
					<button type="submit" disabled={resendState === "sending"} className="btn btn-cta w-full">
						{resendState === "sending" ? "Enviando…" : "Reenviar link"}
					</button>
				</form>
			)}
		</div>
	);
}
