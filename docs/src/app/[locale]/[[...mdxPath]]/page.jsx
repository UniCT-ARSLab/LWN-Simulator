import { generateStaticParamsFor, importPage } from "nextra/pages";
import { useMDXComponents } from "@/mdx-components";

const generateLocalizedParams = generateStaticParamsFor("mdxPath", "locale");

export async function generateStaticParams() {
  const params = await generateLocalizedParams();

  return params.map((param) => ({
    ...param,
    ...(Array.isArray(param.mdxPath) && param.mdxPath[0] === ""
      ? { mdxPath: undefined }
      : {}),
  }));
}

export async function generateMetadata(props) {
  const params = await props.params;
  const { metadata } = await importPage(params.mdxPath, params.locale);
  return metadata;
}

const Wrapper = useMDXComponents().wrapper;

export default async function Page(props) {
  const params = await props.params;
  const result = await importPage(params.mdxPath, params.locale);
  const { default: MDXContent, toc, metadata } = result;

  return (
    <Wrapper toc={toc} metadata={metadata}>
      <MDXContent {...props} params={params} />
    </Wrapper>
  );
}
