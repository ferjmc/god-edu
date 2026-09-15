import { useEffect, useState, type FormEvent } from "react";
import { resetPassword } from "../lib/api/auth";
import { ApiError } from "../lib/api/client";
import { CheckCircleIcon, XCircleIcon } from "./icons";

type Status = "checking" | "missing" | "form" | "submitting" | "success" | "error";

/** Lee ?token= de la URL (mismo patrón que VerifyEmailStatus: esta página no
 * tiene ruta dinámica en Astro, el token viaja como query string) y consume
 * POST /auth/reset-password. El link viene del email que dispara
 * AuthHandler.sendPasswordResetEmail, siempre a /reset-password?token=... —
 * esa ruta está hardcodeada en el backend, no se puede renombrar acá. */
export default function ResetPasswordForm() {
	const [status, setStatus] = useState<Status>("checking");
	const [token, setToken] = useState<string | null>(null);
	const [password, setPassword] = useState("");
	const [confirmPassword, setConfirmPassword] = useState("");
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		const t = new URLSearchParams(window.location.search).get("token");
		if (!t) {
			setStatus("missing");
			return;
		}
		setToken(t);
		setStatus("form");
	}, []);

	async function handleSubmit(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		if (password !== confirmPassword) {
			setError("Las contraseñas no coinciden.");
			return;
		}
		if (!token) return;

		setError(null);
		setStatus("submitting");
		try {
			await resetPassword(token, password);
			setStatus("success");
		} catch (err) {
			setError(err instanceof ApiError ? err.message : "No se pudo restablecer la contraseña.");
			setStatus("error");
		}
	}

	if (status === "checking") {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Verificando" />;
	}

	if (status === "missing") {
		return (
			<div className="flex flex-col items-start gap-3 rounded-[4px] border border-ink-15 bg-paper p-8">
				<XCircleIcon className="h-10 w-10 text-sacred-red" />
				<p className="font-serif text-ink-80">
					Este link no trae ningún token de restablecimiento. Revisá que hayas copiado la URL completa del email.
				</p>
			</div>
		);
	}

	if (status === "success") {
		return (
			<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
				<CheckCircleIcon className="h-10 w-10 text-liturgical-gold" />
				<div>
					<h2 className="font-display text-xl text-marian-blue-deep">Contraseña actualizada</h2>
					<p className="mt-1 font-serif text-ink-80">Ya podés ingresar con tu nueva contraseña.</p>
				</div>
				<a href="/ingresar" className="btn btn-cta">
					Ir a ingresar
				</a>
			</div>
		);
	}

	return (
		<form onSubmit={handleSubmit} className="flex w-full max-w-sm flex-col gap-4">
			<label className="flex flex-col gap-1">
				<span className="font-ui text-sm text-ink-80">Contraseña nueva</span>
				<input
					type="password"
					required
					minLength={8}
					value={password}
					onChange={(e) => setPassword(e.target.value)}
					className="input input-bordered w-full"
				/>
			</label>
			<label className="flex flex-col gap-1">
				<span className="font-ui text-sm text-ink-80">Repetí la contraseña</span>
				<input
					type="password"
					required
					minLength={8}
					value={confirmPassword}
					onChange={(e) => setConfirmPassword(e.target.value)}
					className="input input-bordered w-full"
				/>
			</label>

			{error && (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{error}
				</p>
			)}

			<button type="submit" disabled={status === "submitting"} className="btn btn-primary w-full">
				{status === "submitting" ? "Guardando…" : "Restablecer contraseña"}
			</button>
		</form>
	);
}
