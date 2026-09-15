import { useState, type FormEvent } from "react";
import { forgotPassword } from "../../lib/api/auth";

/** Pide el email y dispara el link de restablecimiento. Igual que el
 * reenvío de verificación (ver VerifyEmailStatus), la API siempre responde
 * el mismo mensaje genérico exista o no la cuenta — no hay nada que
 * distinguir acá, por diseño. */
export default function ForgotPasswordForm() {
	const [email, setEmail] = useState("");
	const [submitting, setSubmitting] = useState(false);
	const [sent, setSent] = useState(false);

	async function handleSubmit(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setSubmitting(true);
		try {
			await forgotPassword(email);
		} finally {
			setSubmitting(false);
			setSent(true);
		}
	}

	if (sent) {
		return (
			<p className="font-serif text-ink-80">
				Si ese email está registrado, te mandamos instrucciones para restablecer tu contraseña.
			</p>
		);
	}

	return (
		<form onSubmit={handleSubmit} className="flex w-full max-w-sm flex-col gap-4">
			<label className="flex flex-col gap-1">
				<span className="font-ui text-sm text-ink-80">Email</span>
				<input
					type="email"
					required
					value={email}
					onChange={(e) => setEmail(e.target.value)}
					className="input input-bordered w-full"
				/>
			</label>

			<button type="submit" disabled={submitting} className="btn btn-primary w-full">
				{submitting ? "Enviando…" : "Enviar instrucciones"}
			</button>
		</form>
	);
}
