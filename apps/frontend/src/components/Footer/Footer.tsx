import { FooterProps } from "./Footer.types";

const Footer = (props: FooterProps) => {
  return (
    <footer className="footer">
      <div className="version">
        BUILD: {process.env.VERSION ? `v${process.env.VERSION}` : "DEV"}
      </div>
      <div className="repo">
        <a
          href="https://github.com/Giuseppe-Compagnone/LWN-Simulator"
          target="_blank"
        >
          Repository
        </a>
      </div>
    </footer>
  );
};

export default Footer;
