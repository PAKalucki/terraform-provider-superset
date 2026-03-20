resource "superset_css_template" "branding" {
  template_name = "Branding"
  css = <<-CSS
    .dashboard {
      background: #f4efe6;
      color: #243447;
    }
  CSS
}
