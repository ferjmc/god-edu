import { useState, type FormEvent } from "react";
import { register } from "../../stores/auth";
import { ApiError } from "../../lib/api/client";

export default function RegisterForm() {
	const [name, setName] = useState("");
	const [email, setEmail] = useState("");
	const [password, setPassword] = useState("");
	const [confirmPassword, setConfirmPassword] = useState("");
	const [acceptedTerms, setAcceptedTerms] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [submitting, setSubmitting] = useState(false);

	async function handleSubmit(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		if (password !== confirmPassword) {
			setError("Las contraseñas no coinciden.");
			return;
		}
		setError(null);
		setSubmitting(true);
		try {
			await register({ name, email, password });
			window.location.assign("/mis-cursos?welcome=1");
		} catch (err) {
			setError(err instanceof ApiError ? err.message : "No se pudo crear la cuenta. Intentá de nuevo.");
			setSubmitting(false);
		}
	}

	return (
		<form onSubmit={handleSubmit} className="flex w-full max-w-sm flex-col gap-4">
			<label className="flex flex-col gap-1">
				<span className="font-ui text-sm text-ink-80">Nombre</span>
				<input
					type="text"
					required
					value={name}
					onChange={(e) => setName(e.target.value)}
					className="input input-bordered w-full"
				/>
			</label>
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
			<label className="flex flex-col gap-1">
				<span className="font-ui text-sm text-ink-80">Contraseña</span>
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

			<label className="flex items-start gap-2">
				<input
					type="checkbox"
					required
					checked={acceptedTerms}
					onChange={(e) => setAcceptedTerms(e.target.checked)}
					className="checkbox mt-0.5"
				/>
				<span className="font-ui text-sm text-ink-80">
					Acepto los{" "}
					<a href="/terminos" target="_blank" className="text-marian-blue hover:text-marian-blue-deep">
						Términos de uso
					</a>{" "}
					y la{" "}
					<a href="/privacidad" target="_blank" className="text-marian-blue hover:text-marian-blue-deep">
						Política de privacidad
					</a>
					.
				</span>
			</label>

			{error && (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{error}
				</p>
			)}

			<button type="submit" disabled={submitting || !acceptedTerms} className="btn btn-primary w-full">
				{submitting ? "Creando cuenta…" : "Crear cuenta"}
			</button>
		</form>
	);
}
