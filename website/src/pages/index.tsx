import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)} style={{ minHeight: '80vh', display: 'flex', alignItems: 'center' }}>
      <div className="container">
        <Heading as="h1" className="hero__title" style={{ fontSize: '4rem', fontWeight: 800 }}>
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle" style={{ fontSize: '1.5rem', opacity: 0.9 }}>{siteConfig.tagline}</p>
        <div className={styles.buttons} style={{ marginTop: '2rem' }}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/intro">
            Get Started
          </Link>
        </div>
      </div>
    </header>
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
        {/* We will build the features grid in Issue #42 */}
      </main>
    </Layout>
  );
}
