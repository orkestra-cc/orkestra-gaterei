import { useEffect, CSSProperties } from 'react';
import L from 'leaflet';
import { MapContainer, Marker, Popup, useMap } from 'react-leaflet';
import MarkerClusterGroup from 'react-leaflet-markercluster';
import 'leaflet.tilelayer.colorfilter';
import 'leaflet/dist/leaflet.css';
import 'react-leaflet-markercluster/styles';
import { useAppContext } from 'providers/AppProvider';

const TILE_URL =
  'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png';
const DARK_FILTER = [
  'invert:98%',
  'grayscale:69%',
  'bright:89%',
  'contrast:111%',
  'hue:205deg',
  'saturate:1000%'
];
const LIGHT_FILTER = [
  'bright:101%',
  'contrast:101%',
  'hue:23deg',
  'saturate:225%'
];

interface MarkerData {
  id: string | number;
  lat: number;
  long: number;
  name: string;
  street: string;
  location: string;
}

interface LayerComponentProps {
  data: MarkerData[];
}

const LayerComponent = ({ data }: LayerComponentProps) => {
  const map = useMap();
  const { config } = useAppContext();
  const { isDark } = config;

  useEffect(() => {
    map.invalidateSize();
  }, [config, map]);

  // Single tile layer with the colorFilter option (leaflet.tilelayer.colorfilter v2 API).
  // Recreated when isDark changes; the cleanup removes the previous layer so we
  // never stack two tile layers on the same map.
  useEffect(() => {
    const layer = L.tileLayer(TILE_URL, {
      colorFilter: isDark ? DARK_FILTER : LIGHT_FILTER
    }).addTo(map);
    return () => {
      map.removeLayer(layer);
    };
  }, [isDark, map]);

  return (
    <MarkerClusterGroup chunkedLoading={true} spiderfyOnMaxZoom={false}>
      {data.map((marker: MarkerData) => (
        <Marker key={marker.id} position={[marker.lat, marker.long]}>
          <Popup>
            <h6 className="mb-1">{marker.name}</h6>
            <p className="m-0 text-500">
              {marker.street} {marker.location}
            </p>
          </Popup>
        </Marker>
      ))}
    </MarkerClusterGroup>
  );
};

interface LeafletMapProps {
  data: MarkerData[];
  className?: string;
  style?: CSSProperties;
}

const LeafletMap = ({ data, className, style }: LeafletMapProps) => {
  const position: [number, number] = [10.737, 0];
  const {
    config: { isRTL }
  } = useAppContext();

  return (
    <MapContainer
      // @ts-expect-error - zoom, minZoom, maxZoom, and zoomSnap are valid MapContainer props from MapOptions
      zoom={isRTL ? 1.8 : 1.5}
      minZoom={isRTL ? 1.8 : 1.3}
      // MarkerClusterGroup mounts before the tile layer (which is added imperatively
      // in a useEffect) and Leaflet throws "Map has no maxZoom specified" if no
      // layer provides one. Setting it on the map itself makes the order irrelevant.
      maxZoom={18}
      zoomSnap={0.5}
      center={position}
      className={className}
      style={style}
    >
      <LayerComponent data={data} />
    </MapContainer>
  );
};

export default LeafletMap;
