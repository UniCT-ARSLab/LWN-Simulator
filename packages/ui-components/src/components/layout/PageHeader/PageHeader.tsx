import { PageHeaderProps } from "./PageHeader.types";

const PageHeader = (props: PageHeaderProps) => {
  return (
    <div className="page-header">
      <div className="titles">
        <div className="title">{props.title}</div>
        {props.subTitle && <div className="sub-title">{props.subTitle}</div>}
      </div>
      <div className="childrens">{props.children}</div>
    </div>
  );
};

export default PageHeader;
