import cn from "classnames";
import { CardLayout, CardProps, CardType } from "./Card.types";
import { forwardRef } from "react";

const Card = forwardRef<HTMLDivElement, CardProps>((props: CardProps, ref) => {
  const layout = props.layout || CardLayout.Default;
  const type = props.type || CardType.Default;

  return (
    <div className={cn("card", props.className, layout, type)} ref={ref}>
      {props.children}
    </div>
  );
});

Card.displayName = "Card";

export default Card;
