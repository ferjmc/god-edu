import { useState, type FormEvent } from "react";
import { register } from "../../stores/auth";
import { ApiError } from "../../lib/api/client";

export default function RegisterForm() {
	const [name, setName] = useState("");
	const [email, setEmail] = useState("");
	const [password, setPassword] = useState("");
	const [error, setError] = useState<string | null>(null);
	const [submitting, setSubmitting] = useState(false);

	async function handleSubmit(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setError(null);
		setSubmitting(true);
		try {
			await register({ name, email, password });
			window.location.assign("/");
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

			{error && (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{error}
				</p>
			)}

			<button type="submit" disabled={submitting} className="btn btn-primary w-full">
				{submitting ? "Creando cuenta…" : "Crear cuenta"}
			</button>
		</form>
	);
}
