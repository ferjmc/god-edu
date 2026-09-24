/** Fecha larga en español rioplatense (ej. "22 de septiembre de 2026"),
 * usada tanto para "Inscripto desde" como para la fecha de emisión de un
 * certificado — mismo formato en ambos lugares, un solo lugar para cambiarlo. */
export function formatLongDate(iso: string): string {
	return new Date(iso).toLocaleDateString("es-AR", { day: "numeric", month: "long", year: "numeric" });
}
