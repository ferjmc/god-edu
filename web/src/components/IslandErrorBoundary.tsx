import { Component, type ErrorInfo, type ReactNode } from "react";

type Props = {
	children: ReactNode;
};

type State = {
	hasError: boolean;
};

/** Atrapa errores de render de una isla de React. Sin esto, cualquier
 * excepción durante el render desmonta la isla entera y deja un hueco en
 * blanco sin ninguna pista. React solo permite error boundaries como
 * componentes de clase. */
export default class IslandErrorBoundary extends Component<Props, State> {
	state: State = { hasError: false };

	static getDerivedStateFromError(): State {
		return { hasError: true };
	}

	componentDidCatch(error: Error, info: ErrorInfo) {
		console.error("IslandErrorBoundary:", error, info.componentStack);
	}

	render() {
		if (!this.state.hasError) return this.props.children;

		return (
			<div role="alert" className="flex flex-col items-start gap-3 rounded-[4px] border border-ink-15 bg-paper p-6">
				<p className="font-serif text-ink-80">Algo salió mal al mostrar esta sección.</p>
				<button type="button" onClick={() => window.location.reload()} className="btn btn-outline btn-sm">
					Recargar
				</button>
			</div>
		);
	}
}
