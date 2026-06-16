import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className="hero" style={{ minHeight: '85vh', display: 'flex', alignItems: 'center', background: 'transparent' }}>
      <div className="container" style={{ textAlign: 'center' }}>
        <Heading as="h1" className="hero__title" style={{ 
          fontSize: 'clamp(3rem, 8vw, 5rem)', 
          fontWeight: 800, 
          letterSpacing: '-0.02em',
          background: 'linear-gradient(135deg, var(--ifm-color-primary) 0%, #E879F9 100%)',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent'
        }}>
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle" style={{ 
          fontSize: 'clamp(1.2rem, 4vw, 1.6rem)', 
          opacity: 0.8, 
          maxWidth: '800px', 
          margin: '0 auto', 
          lineHeight: '1.6' 
        }}>
          {siteConfig.tagline}
        </p>
        <div style={{ marginTop: '3rem', display: 'flex', gap: '1rem', justifyContent: 'center', flexWrap: 'wrap' }}>
          <Link
            className="button button--primary button--lg"
            style={{ borderRadius: '50px', padding: '0.8rem 2rem', fontWeight: 600, boxShadow: '0 10px 25px -5px rgba(139, 92, 246, 0.4)' }}
            to="/docs/intro">
            Get Started
          </Link>
          <Link
            className="button button--secondary button--lg"
            style={{ borderRadius: '50px', padding: '0.8rem 2rem', fontWeight: 600 }}
            href="https://github.com/Vaishnav88sk/claritty">
            View on GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

function FeatureGrid() {
  const features = [
    { title: 'Zero-Trust Local RCA', desc: 'Run state-of-the-art incident analysis entirely on your local Ollama models. Your cluster data never leaves your infrastructure.' },
    { title: 'Universal AI Support', desc: 'Plug and play with Groq, Mistral, OpenAI, or Anthropic. Scale your reasoning engines seamlessly.' },
    { title: 'Decentralized Architecture', desc: 'Designed for Kubernetes. Runs as a lightweight sidecar, requiring zero heavy external dependencies.' }
  ];

  return (
    <section style={{ padding: '4rem 0', background: 'var(--ifm-background-surface-color)' }}>
      <div className="container">
        <div className="row" style={{ display: 'flex', justifyContent: 'center', gap: '2rem', flexWrap: 'wrap' }}>
          {features.map((f, idx) => (
            <div key={idx} style={{
              flex: '1 1 300px',
              padding: '2rem',
              borderRadius: '16px',
              background: 'var(--ifm-background-color)',
              border: '1px solid rgba(139, 92, 246, 0.1)',
              boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
              transition: 'transform 0.2s ease, box-shadow 0.2s ease'
            }}
            onMouseOver={(e) => { e.currentTarget.style.transform = 'translateY(-5px)'; e.currentTarget.style.boxShadow = '0 20px 25px -5px rgba(139, 92, 246, 0.15)'; }}
            onMouseOut={(e) => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.boxShadow = '0 4px 6px -1px rgba(0, 0, 0, 0.1)'; }}
            >
              <Heading as="h3" style={{ fontSize: '1.4rem', color: 'var(--ifm-color-primary)', marginBottom: '1rem' }}>{f.title}</Heading>
              <p style={{ opacity: 0.8, lineHeight: '1.6', margin: 0 }}>{f.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title} | AI-SRE Platform`}
      description="Decentralized AI-SRE observability platform for Kubernetes. Run root cause analysis completely locally with open source LLMs.">
      <HomepageHeader />
      <main>
        <FeatureGrid />
      </main>
    </Layout>
  );
}
