export default function CatalogPage() {
	return (
		<>
			<main className="shell app-shell" data-view="catalog" data-region="eu" aria-live="polite" aria-busy="false" style={{ backgroundColor: "#101820", color: "#f6f8fb", paddingTop: 24, minHeight: "100vh" }}>
				<header className="hero" data-surface="masthead">
					<div className="hero-copy">
						<p className="eyebrow">Spring 2026</p>
						<h1>Catalog</h1>
						<p>Port this layout.</p>
						<nav aria-label="Catalog sections">
							<ul className="hero-links">
								<li><a href="#featured">Featured</a></li>
								<li><a href="#bundles">Bundles</a></li>
								<li><a href="#faq">FAQ</a></li>
							</ul>
						</nav>
					</div>
					<aside className="hero-panel">
						<price-badge data-sku="starter-kit" tone="sale" priority={2} featured>
							<span>Save 20%</span>
						</price-badge>
						<inventory-pill data-state="low">Only 3 left</inventory-pill>
					</aside>
				</header>
				<section id="featured" hidden data-track="featured">
					<article className="card-grid">
						<figure className="media-shell">
							<img src="/hero.png" alt="Hero art" />
							<figcaption>Hero art</figcaption>
						</figure>
						<div className="content-stack">
							<h2>Starter Kit</h2>
							<p>Everything needed to launch a catalog shell.</p>
							<ul className="feature-list">
								<li>Composable sections</li>
								<li>Inspectable markup</li>
								<li>Portable design tokens</li>
							</ul>
							<div className="actions">
								<button type="button" disabled>Sold out</button>
								<input type="checkbox" checked />
								{42}
								<marketing-card featured data-sku="starter-kit" priority={2}>
									<span>Waitlist</span>
									<p>Join to hear when inventory returns.</p>
								</marketing-card>
							</div>
						</div>
					</article>
					<section id="bundles" className="related">
						<h3>Bundles</h3>
						<promo-callout data-tier="plus" tone="contrast">
							<strong>Ships next week</strong>
							<small>Bundle pricing available</small>
						</promo-callout>
					</section>
				</section>
				<footer className="page-footer" data-zone="global">
					<small>Terms apply.</small>
				</footer>
			</main>
		</>
	)
}