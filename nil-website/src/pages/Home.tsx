import { Hero } from "../components/Hero";
import { BenchmarkSection } from "../components/BenchmarkSection";
import { AlgorithmWalkthrough } from "../components/AlgorithmWalkthrough";

export const Home = () => {
  return (
    <>
      <Hero />
      <BenchmarkSection />
      <AlgorithmWalkthrough />
      
      <section className="py-24 bg-primary text-primary-foreground text-center">
        <div className="container mx-auto px-4">
          <h2 className="text-3xl md:text-4xl font-bold mb-6">Ready to verify it yourself?</h2>
          <p className="text-lg mb-8 opacity-90 max-w-xl mx-auto">
            Download our CLI tool to shard, commit, and verify data locally using our Rust implementation.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <button className="bg-background text-foreground px-8 py-4 rounded-full font-bold hover:bg-white/90 transition-colors">
              Get nil-cli
            </button>
            <button className="border border-primary-foreground/30 px-8 py-4 rounded-full font-bold hover:bg-primary-foreground/10 transition-colors">
              View on GitHub
            </button>
          </div>
        </div>
      </section>
    </>
  );
};
