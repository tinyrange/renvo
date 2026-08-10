package apk

const (
	manifestNoIndex       = -1
	manifestTypeString    = 3
	manifestTypeInteger   = 0x10
	manifestTypeBoolean   = 0x12
	manifestTypeReference = 1
)

type manifestPool struct {
	values []string
}

type manifestAttribute struct {
	namespace int
	name      int
	raw       int
	kind      int
	value     int
}

func (pool *manifestPool) add(value string) int {
	for i := 0; i < len(pool.values); i++ {
		if pool.values[i] == value {
			return i
		}
	}
	pool.values = append(pool.values, value)
	return len(pool.values) - 1
}

func buildManifest(config Config) []byte {
	pool := new(manifestPool)
	resourceNames := []string{
		"versionCode", "versionName", "minSdkVersion", "targetSdkVersion",
		"label", "hasCode", "extractNativeLibs", "name", "exported",
		"theme", "value", "screenOrientation",
	}
	resourceIDs := []int{
		0x0101021b, 0x0101021c, 0x0101020c, 0x01010270,
		0x01010001, 0x0101000c, 0x010104ea, 0x01010003, 0x01010010,
		0x01010000, 0x01010024, 0x0101001e,
	}
	for i := 0; i < len(resourceNames); i++ {
		pool.add(resourceNames[i])
	}
	manifestName := pool.add("manifest")
	packageName := pool.add("package")
	usesSDKName := pool.add("uses-sdk")
	applicationName := pool.add("application")
	activityName := pool.add("activity")
	metadataName := pool.add("meta-data")
	intentFilterName := pool.add("intent-filter")
	actionName := pool.add("action")
	categoryName := pool.add("category")
	androidPrefix := pool.add("android")
	androidNamespace := pool.add("http://schemas.android.com/apk/res/android")
	packageValue := pool.add(config.Package)
	versionValue := pool.add(config.VersionName)
	labelValue := pool.add(config.Name)
	nativeActivityValue := pool.add("android.app.NativeActivity")
	libMetadataName := pool.add("android.app.lib_name")
	libMetadataValue := pool.add("renvo")
	mainActionValue := pool.add("android.intent.action.MAIN")
	launcherCategoryValue := pool.add("android.intent.category.LAUNCHER")

	body := make([]byte, 0, 2048)
	body = manifestNamespace(body, 0x0100, androidPrefix, androidNamespace)
	body = manifestStartElement(body, manifestName, []manifestAttribute{
		manifestStringAttribute(manifestNoIndex, packageName, packageValue),
		manifestIntegerAttribute(androidNamespace, 0, config.VersionCode),
		manifestStringAttribute(androidNamespace, 1, versionValue),
	})
	body = manifestStartElement(body, usesSDKName, []manifestAttribute{
		manifestIntegerAttribute(androidNamespace, 2, config.MinSDK),
		manifestIntegerAttribute(androidNamespace, 3, config.TargetSDK),
	})
	body = manifestEndElement(body, usesSDKName)
	body = manifestStartElement(body, applicationName, []manifestAttribute{
		manifestStringAttribute(androidNamespace, 4, labelValue),
		manifestBooleanAttribute(androidNamespace, 5, false),
		manifestBooleanAttribute(androidNamespace, 6, true),
		manifestReferenceAttribute(androidNamespace, 9, 0x01030007),
	})
	activityAttributes := []manifestAttribute{
		manifestStringAttribute(androidNamespace, 7, nativeActivityValue),
		manifestStringAttribute(androidNamespace, 4, labelValue),
		manifestBooleanAttribute(androidNamespace, 8, true),
	}
	if config.Orientation == "portrait" {
		activityAttributes = append(activityAttributes, manifestIntegerAttribute(androidNamespace, 11, 1))
	} else if config.Orientation == "landscape" {
		activityAttributes = append(activityAttributes, manifestIntegerAttribute(androidNamespace, 11, 0))
	}
	body = manifestStartElement(body, activityName, activityAttributes)
	body = manifestStartElement(body, metadataName, []manifestAttribute{
		manifestStringAttribute(androidNamespace, 7, libMetadataName),
		manifestStringAttribute(androidNamespace, 10, libMetadataValue),
	})
	body = manifestEndElement(body, metadataName)
	body = manifestStartElement(body, intentFilterName, nil)
	body = manifestStartElement(body, actionName, []manifestAttribute{
		manifestStringAttribute(androidNamespace, 7, mainActionValue),
	})
	body = manifestEndElement(body, actionName)
	body = manifestStartElement(body, categoryName, []manifestAttribute{
		manifestStringAttribute(androidNamespace, 7, launcherCategoryValue),
	})
	body = manifestEndElement(body, categoryName)
	body = manifestEndElement(body, intentFilterName)
	body = manifestEndElement(body, activityName)
	body = manifestEndElement(body, applicationName)
	body = manifestEndElement(body, manifestName)
	body = manifestNamespace(body, 0x0101, androidPrefix, androidNamespace)

	strings := manifestStringPool(pool.values)
	resources := make([]byte, 0, 8+len(resourceIDs)*4)
	resources = append16(resources, 0x0180)
	resources = append16(resources, 8)
	resources = append32(resources, 8+len(resourceIDs)*4)
	for i := 0; i < len(resourceIDs); i++ {
		resources = append32(resources, resourceIDs[i])
	}
	out := make([]byte, 0, 8+len(strings)+len(resources)+len(body))
	out = append16(out, 0x0003)
	out = append16(out, 8)
	out = append32(out, 8+len(strings)+len(resources)+len(body))
	out = append(out, strings...)
	out = append(out, resources...)
	return append(out, body...)
}

func manifestStringAttribute(namespace int, name int, value int) manifestAttribute {
	return manifestAttribute{namespace: namespace, name: name, raw: value,
		kind: manifestTypeString, value: value}
}

func manifestIntegerAttribute(namespace int, name int, value int) manifestAttribute {
	return manifestAttribute{namespace: namespace, name: name, raw: manifestNoIndex,
		kind: manifestTypeInteger, value: value}
}

func manifestBooleanAttribute(namespace int, name int, value bool) manifestAttribute {
	encoded := 0
	if value {
		encoded = -1
	}
	return manifestAttribute{namespace: namespace, name: name, raw: manifestNoIndex,
		kind: manifestTypeBoolean, value: encoded}
}

func manifestReferenceAttribute(namespace int, name int, value int) manifestAttribute {
	return manifestAttribute{namespace: namespace, name: name, raw: manifestNoIndex,
		kind: manifestTypeReference, value: value}
}

func manifestStringPool(values []string) []byte {
	offsets := make([]int, len(values))
	data := make([]byte, 0, len(values)*16)
	for i := 0; i < len(values); i++ {
		offsets[i] = len(data)
		length := len(values[i])
		data = append(data, byte(length), byte(length))
		data = append(data, []byte(values[i])...)
		data = append(data, 0)
	}
	for len(data)%4 != 0 {
		data = append(data, 0)
	}
	stringsStart := 28 + len(offsets)*4
	out := make([]byte, 0, stringsStart+len(data))
	out = append16(out, 0x0001)
	out = append16(out, 28)
	out = append32(out, stringsStart+len(data))
	out = append32(out, len(values))
	out = append32(out, 0)
	out = append32(out, 0x100)
	out = append32(out, stringsStart)
	out = append32(out, 0)
	for i := 0; i < len(offsets); i++ {
		out = append32(out, offsets[i])
	}
	return append(out, data...)
}

func manifestNamespace(out []byte, kind int, prefix int, namespace int) []byte {
	out = append16(out, kind)
	out = append16(out, 16)
	out = append32(out, 24)
	out = append32(out, 1)
	out = append32(out, manifestNoIndex)
	out = append32(out, prefix)
	return append32(out, namespace)
}

func manifestStartElement(
	out []byte, name int, attributes []manifestAttribute,
) []byte {
	size := 36 + len(attributes)*20
	out = append16(out, 0x0102)
	out = append16(out, 16)
	out = append32(out, size)
	out = append32(out, 1)
	out = append32(out, manifestNoIndex)
	out = append32(out, manifestNoIndex)
	out = append32(out, name)
	out = append16(out, 20)
	out = append16(out, 20)
	out = append16(out, len(attributes))
	out = append16(out, 0)
	out = append16(out, 0)
	out = append16(out, 0)
	for i := 0; i < len(attributes); i++ {
		attribute := attributes[i]
		out = append32(out, attribute.namespace)
		out = append32(out, attribute.name)
		out = append32(out, attribute.raw)
		out = append16(out, 8)
		out = append(out, 0, byte(attribute.kind))
		out = append32(out, attribute.value)
	}
	return out
}

func manifestEndElement(out []byte, name int) []byte {
	out = append16(out, 0x0103)
	out = append16(out, 16)
	out = append32(out, 24)
	out = append32(out, 1)
	out = append32(out, manifestNoIndex)
	out = append32(out, manifestNoIndex)
	return append32(out, name)
}
