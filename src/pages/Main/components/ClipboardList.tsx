import { Virtuoso } from "react-virtuoso";

import { useClipboardItems } from "../hooks/useClipboardItems";
import ClipboardCard from "./cards/ClipboardCard";

const ClipboardList = () => {
  const { items, loadMore, actions } = useClipboardItems();

  return (
    <Virtuoso
      computeItemKey={(_, item) => item.id}
      data={items}
      endReached={loadMore}
      increaseViewportBy={200}
      itemContent={(_, item) => <ClipboardCard actions={actions} item={item} />}
      style={{ height: "100%" }}
    />
  );
};

export default ClipboardList;
