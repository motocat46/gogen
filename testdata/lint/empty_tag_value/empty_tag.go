package empty_tag_value

// EmptyTagValue: gogen tag 存在但值为空字符串，应被视为无注解，不产生任何 issue。
type EmptyTagValue struct {
	Name string `gogen:""`
}
