import { useEffect, useState, type FormEvent } from "react";
import { login } from "../../stores/auth";
import { ApiError } from "../../lib/api/client";

export default function LoginForm() {
	const [email, setEmail] = useState("");
	const [password, setPassword] = useState("");
	const [error, setError] = useState<string | null>(null);
	const [submitting, setSubmitting] = useState(false);

	// The API redirects here with ?error=oauth when a Google/Facebook login
	// fails (see OAuthHandler.Callback in the Go API) — surface it instead
	// of leaving the user on a silently blank form.
	useEffect(() => {
		if (new URLSearchParams(window.location.search).get("error") === "oauth") {
			setError("No pudimos completar el ingreso con ese proveedor. Probá de nuevo.");
		}
	}, []);

	async function handleSubmit(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setError(null);
		setSubmitting(true);
		try {
			await login({ email, password });
			window.location.assign("/mis-cursos");
		} catch (err) {
			setError(err instanceof ApiError ? err.message : "No se pudo iniciar sesión. Intentá de nuevo.");
			setSubmitting(false);
		}
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
			<label className="flex flex-col gap-1">
				<span className="font-ui text-sm text-ink-80">Contraseña</span>
				<input
					type="password"
					required
					value={password}
					onChange={(e) => setPassword(e.target.value)}
					className="input input-bordered w-full"
				/>
				<a href="/olvide-mi-contrasena" className="self-end font-ui text-xs text-marian-blue hover:text-marian-blue-deep">
					¿Olvidaste tu contraseña?
				</a>
			</label>

			{error && (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{error}
				</p>
			)}

			<button type="submit" disabled={submitting} className="btn btn-primary w-full">
				{submitting ? "Ingresando…" : "Ingresar"}
			</button>
		</form>
	);
}
