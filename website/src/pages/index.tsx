import React, { useState, useEffect, type ReactNode } from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';
import Head from '@docusaurus/Head';

function LatestReleaseBadge() {
  const [version, setVersion] = useState<string | null>(null);
  const [url, setUrl] = useState<string>('https://github.com/Vaishnav88sk/claritty/releases');

  useEffect(() => {
    fetch('https://api.github.com/repos/Vaishnav88sk/claritty/releases/latest')
      .then(res => res.json())
      .then(data => {
        if (data && data.tag_name) {
          setVersion(data.tag_name);
          setUrl(data.html_url);
        }
      })
      .catch(console.error);
  }, []);

  if (!version) return null;

  return (
    <div className="animate-fade-in-up" style={{ marginBottom: '1rem', display: 'flex', justifyContent: 'center' }}>
      <a href={url} target="_blank" rel="noopener noreferrer" className="animate-jiggle" style={{
        display: 'inline-flex',
        alignItems: 'center',
        padding: '0.4rem 1rem',
        borderRadius: '50px',
        background: 'rgba(139, 92, 246, 0.1)',
        border: '1px solid rgba(139, 92, 246, 0.2)',
        color: 'var(--ifm-color-primary)',
        fontSize: '0.9rem',
        fontWeight: 600,
        textDecoration: 'none',
        transition: 'all 0.2s ease',
      }}
      onMouseEnter={(e) => { e.currentTarget.style.background = 'rgba(139, 92, 246, 0.2)'; }}
      onMouseLeave={(e) => { e.currentTarget.style.background = 'rgba(139, 92, 246, 0.1)'; }}
      onFocus={(e) => { e.currentTarget.style.background = 'rgba(139, 92, 246, 0.2)'; }}
      onBlur={(e) => { e.currentTarget.style.background = 'rgba(139, 92, 246, 0.1)'; }}
      >
        <span style={{ marginRight: '8px' }}>🚀</span>
        Claritty {version} is now available!
      </a>
    </div>
  );
}

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className="hero" style={{ minHeight: '85vh', display: 'flex', alignItems: 'center', background: 'transparent', padding: '6rem 0' }}>
      <div className="container" style={{ textAlign: 'center' }}>
        
        {/* Dynamic Latest Release Badge */}
        <LatestReleaseBadge />

        {/* Animated Headline */}
        <Heading as="h1" className="hero__title animate-fade-in-up" style={{ 
          fontSize: 'clamp(3rem, 8vw, 5rem)', 
          fontWeight: 800, 
          letterSpacing: '-0.02em',
          background: 'linear-gradient(135deg, var(--ifm-color-primary) 0%, #E879F9 100%)',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent'
        }}>
          {siteConfig.title}
        </Heading>
        
        {/* Animated Subtitle */}
        <p className="hero__subtitle animate-fade-in-up delay-1" style={{ 
          fontSize: 'clamp(1.2rem, 4vw, 1.6rem)', 
          opacity: 0.8, 
          maxWidth: '800px', 
          margin: '0 auto', 
          lineHeight: '1.6' 
        }}>
          {siteConfig.tagline}
        </p>
        
        {/* Animated Buttons */}
        <div className="animate-fade-in-up delay-2" style={{ marginTop: '3rem', display: 'flex', gap: '1rem', justifyContent: 'center', flexWrap: 'wrap' }}>
          <Link
            className="button button--primary button--lg"
            style={{ borderRadius: '50px', padding: '0.8rem 2rem', fontWeight: 600, boxShadow: '0 10px 25px -5px rgba(139, 92, 246, 0.4)', transition: 'transform 0.2s ease, box-shadow 0.2s ease' }}
            to="/docs/intro"
            onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-2px)'; e.currentTarget.style.boxShadow = '0 15px 30px -5px rgba(139, 92, 246, 0.6)'; }}
            onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.boxShadow = '0 10px 25px -5px rgba(139, 92, 246, 0.4)'; }}
            onFocus={(e) => { e.currentTarget.style.transform = 'translateY(-2px)'; e.currentTarget.style.boxShadow = '0 15px 30px -5px rgba(139, 92, 246, 0.6)'; }}
            onBlur={(e) => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.boxShadow = '0 10px 25px -5px rgba(139, 92, 246, 0.4)'; }}
            >
            Get Started
          </Link>
          <Link
            className="button button--secondary button--lg"
            style={{ borderRadius: '50px', padding: '0.8rem 2rem', fontWeight: 600, transition: 'transform 0.2s ease' }}
            href="https://github.com/Vaishnav88sk/claritty"
            onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-2px)'; }}
            onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)'; }}
            onFocus={(e) => { e.currentTarget.style.transform = 'translateY(-2px)'; }}
            onBlur={(e) => { e.currentTarget.style.transform = 'translateY(0)'; }}
            >
            View on GitHub
          </Link>
        </div>

        {/* Visual Demo (Terminal Mockup) */}
        <div className="animate-fade-in-up delay-3" style={{ marginTop: '5rem', display: 'flex', justifyContent: 'center' }}>
          <div className="animate-float" style={{
            maxWidth: '900px',
            width: '100%',
            borderRadius: '12px',
            overflow: 'hidden',
            boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(139, 92, 246, 0.2)',
            background: 'var(--ifm-background-surface-color)'
          }}>
            {/* Fake Mac Window Controls */}
            <div style={{ height: '30px', background: 'rgba(0,0,0,0.2)', display: 'flex', alignItems: 'center', padding: '0 1rem', gap: '6px' }}>
              <div style={{ width: '12px', height: '12px', borderRadius: '50%', background: '#FF5F56' }}></div>
              <div style={{ width: '12px', height: '12px', borderRadius: '50%', background: '#FFBD2E' }}></div>
              <div style={{ width: '12px', height: '12px', borderRadius: '50%', background: '#27C93F' }}></div>
            </div>
            {/* Terminal Image */}
            <img 
              src="img/claritty-clarctl-1.png" 
              alt="Claritty CLI Terminal Interface" 
              style={{ width: '100%', display: 'block' }}
            />
          </div>
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
        <div className="row animate-fade-in-up delay-4" style={{ display: 'flex', justifyContent: 'center', gap: '2rem', flexWrap: 'wrap' }}>
          {features.map((f, idx) => (
            <div key={idx} style={{
              flex: '1 1 300px',
              padding: '2rem',
              borderRadius: '16px',
              background: 'var(--ifm-background-color)',
              border: '1px solid rgba(139, 92, 246, 0.1)',
              boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
              transition: 'transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s cubic-bezier(0.4, 0, 0.2, 1)'
            }}
            onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-8px) scale(1.02)'; e.currentTarget.style.boxShadow = '0 20px 25px -5px rgba(139, 92, 246, 0.15), 0 0 0 1px rgba(139, 92, 246, 0.3)'; }}
            onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0) scale(1)'; e.currentTarget.style.boxShadow = '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)'; }}
            onFocus={(e) => { e.currentTarget.style.transform = 'translateY(-8px) scale(1.02)'; e.currentTarget.style.boxShadow = '0 20px 25px -5px rgba(139, 92, 246, 0.15), 0 0 0 1px rgba(139, 92, 246, 0.3)'; }}
            onBlur={(e) => { e.currentTarget.style.transform = 'translateY(0) scale(1)'; e.currentTarget.style.boxShadow = '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)'; }}
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
      description="Decentralized AI-SRE observability platform for Kubernetes. Run root cause analysis completely locally with open source LLMs.">
      <Head>
        <title>Claritty | AI-SRE Platform for Kubernetes</title>
        <meta property="og:title" content="Claritty | AI-SRE Platform for Kubernetes" />
      </Head>
      <HomepageHeader />
      <main>
        <FeatureGrid />
      </main>
    </Layout>
  );
}
