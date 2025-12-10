import { useState, useEffect, type ReactNode } from 'react';
import { motion } from 'framer-motion';
import { marked } from 'marked';

interface MarkdownPageProps {
  filePath: string;
  title: string;
}

const MarkdownPage = ({ filePath, title }: MarkdownPageProps) => {
  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchMarkdown = async () => {
      try {
        const response = await fetch(filePath);
        if (!response.ok) {
          throw new Error(`Failed to fetch ${filePath}: ${response.statusText}`);
        }
        const text = await response.text();
        setContent(marked.parse(text));
      } catch (err) {
        const message = err instanceof Error ? err.message : 'An unexpected error occurred';
        setError(message);
      } finally {
        setLoading(false);
      }
    };

    fetchMarkdown();
  }, [filePath]);

  const renderPage = (body: ReactNode) => (
    <div className="bg-gradient-to-b from-background via-secondary/10 to-background w-full">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-16 space-y-8"
      >
        <div className="space-y-3">
          <p className="text-xs font-semibold uppercase tracking-[0.28em] text-primary/80">Research</p>
          <h1 className="text-4xl sm:text-5xl font-bold tracking-tight text-foreground">{title}</h1>
          <p className="text-lg text-muted-foreground max-w-3xl">
            A formatted reading experience for the latest NilStore papers with consistent spacing, contrast, and typography.
          </p>
        </div>

        <div className="bg-card border border-border/60 rounded-2xl shadow-lg shadow-black/5 p-6 sm:p-8 lg:p-10">
          {body}
        </div>
      </motion.div>
    </div>
  );

  if (loading) return renderPage(<p className="text-muted-foreground">Loading...</p>);

  if (error) return renderPage(<p className="text-destructive">Error: {error}</p>);

  return renderPage(
    <div
      className="markdown-content"
      dangerouslySetInnerHTML={{ __html: content }}
    />
  );
};

export const Litepaper = () => <MarkdownPage filePath="/litepaper.md" title="NilStore Litepaper" />;
export const Whitepaper = () => <MarkdownPage filePath="/whitepaper.md" title="NilStore Whitepaper" />;
