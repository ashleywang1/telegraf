// Package tableprov_csv implements an Telegraf serializer for Goblin.
package tableprov_csv

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/influxdata/telegraf"
	pb "goblin.dde.akamai.com/generated/grpc/goblin_common"
	ingestPb "goblin.dde.akamai.com/generated/grpc/goblin_ingest"
)

type TableprovCSVSerializer struct {
	HostIP string
}

func (s *TableprovCSVSerializer) SerializeBatch(metrics []telegraf.Metric) ([]byte, error) {
	var batch bytes.Buffer
	for _, m := range metrics {

		buf, err := s.Serialize(m)

		if err != nil {
			return nil, err
		}

		_, err = batch.Write(buf)

		if err != nil {
			return nil, err
		}
	}

	return batch.Bytes(), nil

}

func (s *TableprovCSVSerializer) Serialize(m telegraf.Metric) ([]byte, error) {
	var b bytes.Buffer
	buf, err := s.serializeData(m)
	if err != nil {
		return nil, err
	}
	chunkNumber := 0 // default, will change later
	isLast := false  // default, may change later
	if m.HasField("tableprov") && m.HasTag("chunkNumber") {
		chunkNumber, err = strconv.Atoi(m.Tags()["chunkNumber"])
		if err != nil {
			return nil, err
		}
	}
	if m.HasField("tableprov") && m.HasTag("isLast") {
		isLast, err = strconv.ParseBool(m.Tags()["isLast"])
		if err != nil {
			return nil, err
		}
	} else {
		isLast = true
	}

	identifier := s.createIdentifier(m, uint32(chunkNumber), isLast)
	properties := s.createProperties(true)
	if m.Name() == "alert" || m.Name() == "alerts2" {
		properties = s.createProperties(false)
	}

	table := pb.TableSnapshotChunk{
		Version:            pb.TableSnapshotChunkVersion_TABLE_SNAPSHOT_CHUNK_VERSION_0_1,
		ChunkIdentifier:    identifier,
		SnapshotProperties: properties,
		Data:               buf,
	}

	publishChunk := ingestPb.PublishTableChunk{
		Chunk: &table,
	}

	serialized, err := proto.Marshal(&publishChunk)
	if err != nil {
		return nil, err
	}

	// Add self-delimiting data, a 4-byte Integer denoting how large the data is
	// header := make([]byte, 4)
	// binary.BigEndian.PutUint32(header, uint32(len(serialized)))
	// b.Write(header)
	b.Write(serialized)

	return b.Bytes(), nil
}

func (s *TableprovCSVSerializer) serializeData(m telegraf.Metric) ([]byte, error) {
	var b bytes.Buffer

	// Deal with the special case where the metric is from the tableprov input plugin
	if tableprovMetric, ok := m.Fields()["tableprov"]; ok {
		metricString, isString := tableprovMetric.(string)
		if !isString {
			return nil, fmt.Errorf("Cannot serialize the tableprov metric of type %T", tableprovMetric)
		}
		b.WriteString(metricString)
		return b.Bytes(), nil
	}

	// Generate the serialization for all other metrics
	// use the timestamp as the version indicator
	timestamp := strconv.FormatInt(m.Time().Unix(), 10)
	b.WriteString(timestamp + "\n")

	// Sort the tags to get a reliable order
	var tagKeys []string
	for k := range m.Tags() {
		tagKeys = append(tagKeys, k)
	}
	sort.Strings(tagKeys)

	for _, k := range tagKeys {
		tagValue, err := m.GetTag(k)
		if err == false {
			return b.Bytes(), fmt.Errorf("tag does not exist %s", k)
		}
		b.WriteString(k + "=" + tagValue + ",")
	}

	b.Truncate(b.Len() - 1)
	b.WriteString("\n")

	// Sort keys first to maintain reliable order
	var keys []string
	for k := range m.Fields() {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var colNames bytes.Buffer
	var colTypes bytes.Buffer
	var fields bytes.Buffer

	for _, k := range keys {
		colNames.WriteString(k + ",")
		vType, value, err := mapTelegrafToTableprovTypes(m.GetField(k))
		if err != nil {
			return b.Bytes(), err
		}
		colTypes.WriteString(vType + ",")
		fields.WriteString(value + ",")
	}
	// We cannot have empty metric fields
	colNames.Truncate(colNames.Len() - 1)
	colTypes.Truncate(colTypes.Len() - 1)
	fields.Truncate(fields.Len() - 1)
	colNames.WriteString("\n")
	colTypes.WriteString("\n")
	fields.WriteString("\n")
	b.Write(colNames.Bytes())
	b.Write(colTypes.Bytes())
	b.Write(colNames.Bytes()) // column descriptions
	b.Write(fields.Bytes())

	return b.Bytes(), nil
}

func (s *TableprovCSVSerializer) createIdentifier(m telegraf.Metric, n uint32, isLast bool) *pb.TableSnapshotChunkIdentifier {
	return &pb.TableSnapshotChunkIdentifier{
		SnapshotIdentifier: &pb.TableSnapshotIdentifier{
			TableIdentifier: &pb.TableIdentifier{
				NetworkName: "infra",
				TableName:   m.Name(),
			},
			PublisherIdentifier: &pb.PublisherIdentifier{
				Type:       pb.PublisherIdentifier_IPv4,
				Identifier: s.HostIP,
			},
			PublicationTimestamp: &pb.Timestamp{
				Time: uint64(m.Time().Unix()),
			},
		},
		ChunkSequenceNumber: n,
		IsLastInSnapshot:    isLast,
	}
}

func (s *TableprovCSVSerializer) createProperties(replace bool) *pb.TableSnapshotProperties {
	aggType := pb.SnapshotWindowingSemantics_REPLACE
	if !replace {
		aggType = pb.SnapshotWindowingSemantics_APPEND
	}
	return &pb.TableSnapshotProperties{
		IsPresent: true,
		EncodingMetadata: &pb.TableEncodingMetadata{
			Encoding: pb.TableEncoding_TABLEPROV_CSV,
		},
		WindowingSemantics: aggType,
	}
}

func mapTelegrafToTableprovTypes(v interface{}, is_field bool) (string, string, error) {

	if !(is_field) {
		return "", "", fmt.Errorf("No such field in metric of type %s", v)
	}

	switch x := v.(type) {
	case float64:
		return "int", strconv.FormatInt(int64(x), 10), nil
	case float32:
		return "int", strconv.FormatInt(int64(x), 10), nil
	case int64:
		return "int", strconv.FormatInt(x, 10), nil
	case string:
		return "string", x, nil
	case bool:
		return "string", strconv.FormatBool(x), nil
	case int32:
		return "int", strconv.FormatInt(int64(x), 10), nil
	case int16:
		return "int", strconv.FormatInt(int64(x), 10), nil
	case int8:
		return "int", strconv.FormatInt(int64(x), 10), nil
	case int:
		return "int", strconv.FormatInt(int64(x), 10), nil
	case uint64:
		return "int", strconv.FormatUint(x, 10), nil
	case uint32:
		return "int", strconv.FormatUint(uint64(x), 10), nil
	case uint16:
		return "int", strconv.FormatUint(uint64(x), 10), nil
	case uint8:
		return "int", strconv.FormatUint(uint64(x), 10), nil
	case uint:
		return "int", strconv.FormatUint(uint64(x), 10), nil
	case []byte:
		return "string", string(x), nil
	default:
		return "string", fmt.Sprintf("%s", v), nil
	}
}
