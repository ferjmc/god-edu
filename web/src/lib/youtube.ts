/**
 * Extrae el video ID de una URL de YouTube en cualquiera de sus formatos
 * comunes (watch?v=, youtu.be/, /embed/) y arma la URL de embed. Usa
 * youtube-nocookie.com — modo de privacidad ampliada de YouTube, no manda
 * cookies de tracking hasta que el usuario interactúa con el video.
 *
 * Devuelve null si la URL no matchea ningún formato conocido, en vez de
 * tirar: un dato mal cargado no debería romper el render de la lección.
 */
export function toYoutubeEmbedUrl(url: string): string | null {
	let id: string | null = null;

	try {
		const parsed = new URL(url);
		if (parsed.hostname === "youtu.be") {
			id = parsed.pathname.slice(1);
		} else if (parsed.hostname.endsWith("youtube.com") || parsed.hostname.endsWith("youtube-nocookie.com")) {
			if (parsed.pathname === "/watch") {
				id = parsed.searchParams.get("v");
			} else if (parsed.pathname.startsWith("/embed/")) {
				id = parsed.pathname.replace("/embed/", "");
			}
		}
	} catch {
		return null;
	}

	return id ? `https://www.youtube-nocookie.com/embed/${id}` : null;
}
