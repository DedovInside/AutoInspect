from ultralytics.data.converter import convert_coco

convert_coco(
    labels_dir="../images/CarDD_COCO/annotations",
    save_dir="../images/CarDD_YOLO",
    use_segments=True,
    cls91to80=False,
)