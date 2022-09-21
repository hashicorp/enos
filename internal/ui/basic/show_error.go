package basic

// ShowError writes the error message to stdout
func (v *View) ShowError(err error) error {
	if err == nil {
		return nil
	}
	v.ui.Error(err.Error())
	return nil
}
