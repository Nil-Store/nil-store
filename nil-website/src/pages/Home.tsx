import { Hero } from "../components/Hero";
import { BenchmarkSection } from "../components/BenchmarkSection";
import { Link } from "react-router-dom";
import { ArrowRight, Layers } from "lucide-react";

export const Home = () => {
  return (
    <>
      <Hero />
      <BenchmarkSection />
      
      <section className="py-24 bg-secondary/20">
        <div className="container mx-auto px-4 text-center">
          <div className="max-w-3xl mx-auto">
            <div className="w-16 h-16 bg-primary/10 rounded-2xl flex items-center justify-center mx-auto mb-6">
              <Layers className="w-8 h-8 text-primary" />
            </div>
            <h2 className="text-3xl font-bold mb-4">Understanding the Protocol</h2>
            <p className="text-muted-foreground mb-8 text-lg">
              NilStore is built on a novel combination of Sharding, KZG Commitments, and Proof-of-Seal. 
              We've built interactive visualizations to explain exactly how your data is secured.
            </p>
            <Link 
              to="/technology" 
              className="inline-flex items-center gap-2 px-8 py-4 bg-card border hover:bg-secondary/50 rounded-full font-medium transition-all"
            >
              Explore the Technology <ArrowRight className="w-4 h-4" />
            </Link>
          </div>
        </div>
      </section>
      
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
