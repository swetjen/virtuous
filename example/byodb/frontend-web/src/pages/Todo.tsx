export function Todo({ title = "Coming soon" }: { title?: string }) {
  return (
    <section className="panel">
      <h2>{title}</h2>
      <p>This section is scaffolded so teams can add domain logic without changing route structure.</p>
    </section>
  );
}
