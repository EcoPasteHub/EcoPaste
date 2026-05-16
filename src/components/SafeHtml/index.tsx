import DOMPurify from "dompurify";
import { type FC, type MouseEvent, useContext } from "react";
import { Marker } from "react-mark.js";
import { MainContext } from "@/pages/Main";

interface SafeHtmlProps {
  value: string;
}

const SafeHtml: FC<SafeHtmlProps> = (props) => {
  const { value } = props;
  const { rootState } = useContext(MainContext);

  const handleClick = (event: MouseEvent) => {
    const { target, metaKey, ctrlKey } = event;

    const link = (target as HTMLElement).closest("a");

    if (!link || metaKey || ctrlKey) return;

    event.preventDefault();
    event.stopPropagation();
  };

  return (
    <Marker mark={rootState.search}>
      <div
        className="translate-z-0"
        dangerouslySetInnerHTML={{
          __html: DOMPurify.sanitize(value, {
            FORBID_ATTR: ["target", "controls", "autoplay", "autoPlay"],
          }),
        }}
        onClick={handleClick}
      />
    </Marker>
  );
};

export default SafeHtml;
